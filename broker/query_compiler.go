//  Copyright (c) 2017-2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	memCom "github.com/uber/aresdb/memstore/common"
	metaCom "github.com/uber/aresdb/metastore/common"
	"github.com/uber/aresdb/query/common"
	"github.com/uber/aresdb/query/expr"
	"github.com/uber/aresdb/utils"
	"net/http"
	"strconv"
	"strings"
)

const (
	nonAggregationQueryLimit = 1000
)

// QueryContext is broker query context
type QueryContext struct {
	AQLQuery              *common.AQLQuery
	IsNonAggregationQuery bool
	ReturnHLLBinary       bool
	Writer                http.ResponseWriter
	Error                 error
	Tables                []*memCom.TableSchema
	TableIDByAlias        map[string]int
	TableSchemaByName     map[string]*memCom.TableSchema

	NumDimsPerDimWidth common.DimCountsPerDimWidth
	// lookup table from enum dimension index to EnumDict, used for postprocessing
	DimensionEnumReverseDicts map[int][]string
	// this should be the same as generated by datanodes. in the future we should pass
	// it down to datanodes
	DimensionVectorIndex []int
	DimRowBytes          int
	RequestID            string
}

// NewQueryContext creates new query context
func NewQueryContext(aql *common.AQLQuery, returnHLLBinary bool, w http.ResponseWriter) *QueryContext {
	ctx := QueryContext{
		AQLQuery:                  aql,
		ReturnHLLBinary:           returnHLLBinary,
		Writer:                    w,
		DimensionEnumReverseDicts: make(map[int][]string),
	}
	return &ctx
}

// GetRewrittenQuery get the rewritten query after query parsing
func (qc *QueryContext) GetRewrittenQuery() common.AQLQuery {
	newQuery := *qc.AQLQuery
	for i, measure := range newQuery.Measures {
		if measure.ExprParsed != nil {
			measure.Expr = measure.ExprParsed.String()
			newQuery.Measures[i] = measure
		}
	}

	for i, join := range newQuery.Joins {
		for j := range join.Conditions {
			if j < len(join.ConditionsParsed) && join.ConditionsParsed[j] != nil {
				join.Conditions[j] = join.ConditionsParsed[j].String()
			}
		}
		newQuery.Joins[i] = join
	}

	for i, dim := range newQuery.Dimensions {
		if dim.ExprParsed != nil {
			dim.Expr = dim.ExprParsed.String()
			newQuery.Dimensions[i] = dim
		}
	}

	for i := range newQuery.Filters {
		if i < len(newQuery.FiltersParsed) && newQuery.FiltersParsed[i] != nil {
			newQuery.Filters[i] = newQuery.FiltersParsed[i].String()
		}
	}

	for i, measure := range newQuery.SupportingMeasures {
		if measure.ExprParsed != nil {
			measure.Expr = measure.ExprParsed.String()
			newQuery.SupportingMeasures[i] = measure
		}
	}

	for i, dim := range newQuery.SupportingDimensions {
		if dim.ExprParsed != nil {
			dim.Expr = dim.ExprParsed.String()
			newQuery.SupportingDimensions[i] = dim
		}
	}
	return newQuery
}

// Compile parses expressions into ast, load schema from schema reader, resolve types,
// and collects meta data needed by post processing
func (qc *QueryContext) Compile(tableSchemaReader memCom.TableSchemaReader) {
	qc.readSchema(tableSchemaReader)
	defer qc.releaseSchema()
	if qc.Error != nil {
		return
	}

	qc.processJoins()
	if qc.Error != nil {
		return
	}

	qc.processMeasures()
	if qc.Error != nil {
		return
	}
	qc.processDimensions()
	if qc.Error != nil {
		return
	}

	qc.processFilters()
	if qc.Error != nil {
		return
	}

	qc.sortDimensionColumns()
	return
}

func (qc *QueryContext) readSchema(tableSchemaReader memCom.TableSchemaReader) {
	qc.Tables = make([]*memCom.TableSchema, 1+len(qc.AQLQuery.Joins))
	qc.TableIDByAlias = make(map[string]int)
	qc.TableSchemaByName = make(map[string]*memCom.TableSchema)

	tableSchemaReader.RLock()
	defer tableSchemaReader.RUnlock()

	var (
		err    error
		schema *memCom.TableSchema
	)
	// Main table.
	schema, err = tableSchemaReader.GetSchema(qc.AQLQuery.Table)
	if err != nil {
		qc.Error = utils.StackError(err, "unknown main table %s", qc.AQLQuery.Table)
		return
	}
	qc.TableSchemaByName[qc.AQLQuery.Table] = schema
	schema.RLock()
	qc.Tables[0] = schema

	qc.TableIDByAlias[qc.AQLQuery.Table] = 0

	// Foreign tables.
	for i, join := range qc.AQLQuery.Joins {
		schema, err = tableSchemaReader.GetSchema(join.Table)
		if err != nil {
			qc.Error = utils.StackError(err, "unknown join table %s", join.Table)
			return
		}

		if qc.TableSchemaByName[join.Table] == nil {
			qc.TableSchemaByName[join.Table] = schema
			// Prevent double locking.
			schema.RLock()
		}

		qc.Tables[1+i] = schema

		alias := join.Alias
		if alias == "" {
			alias = join.Table
		}
		_, exists := qc.TableIDByAlias[alias]
		if exists {
			qc.Error = utils.StackError(nil, "table alias %s is redefined", alias)
			return
		}
		qc.TableIDByAlias[alias] = 1 + i
	}
}

func (qc *QueryContext) releaseSchema() {
	for _, schema := range qc.TableSchemaByName {
		schema.RUnlock()
	}
}

func (qc *QueryContext) resolveColumn(identifier string) (int, int, error) {
	tableAlias := qc.AQLQuery.Table
	column := identifier
	segments := strings.SplitN(identifier, ".", 2)
	if len(segments) == 2 {
		tableAlias = segments[0]
		column = segments[1]
	}

	tableID, exists := qc.TableIDByAlias[tableAlias]
	if !exists {
		return 0, 0, utils.StackError(nil, "unknown table alias %s", tableAlias)
	}

	columnID, exists := qc.Tables[tableID].ColumnIDs[column]
	if !exists {
		return 0, 0, utils.StackError(nil, "unknown column %s for table alias %s",
			column, tableAlias)
	}

	return tableID, columnID, nil
}

func (qc *QueryContext) processJoins() {
	var err error
	for i, join := range qc.AQLQuery.Joins {
		join.ConditionsParsed = make([]expr.Expr, len(join.Conditions))
		for j, cond := range join.Conditions {
			join.ConditionsParsed[j], err = expr.ParseExpr(cond)
			if err != nil {
				qc.Error = utils.StackError(err, "Failed to parse join condition: %s", cond)
				return
			}
			join.ConditionsParsed[j] = expr.Rewrite(qc, join.ConditionsParsed[j])
			if qc.Error != nil {
				return
			}
		}
		qc.AQLQuery.Joins[i] = join
	}
}

func (qc *QueryContext) processFilters() {
	var err error

	qc.AQLQuery.FiltersParsed = make([]expr.Expr, len(qc.AQLQuery.Filters))
	for i, filter := range qc.AQLQuery.Filters {
		qc.AQLQuery.FiltersParsed[i], err = expr.ParseExpr(filter)
		if err != nil {
			qc.Error = utils.StackError(err, "Failed to parse filter %s", filter)
			return
		}
		qc.AQLQuery.FiltersParsed[i] = expr.Rewrite(qc, qc.AQLQuery.FiltersParsed[i])
		if qc.Error != nil {
			return
		}
	}

	qc.AQLQuery.FiltersParsed = normalizeAndFilters(qc.AQLQuery.FiltersParsed)
}

func (qc *QueryContext) processMeasures() {
	var err error

	for i, measure := range qc.AQLQuery.Measures {
		measure.ExprParsed, err = expr.ParseExpr(measure.Expr)
		if err != nil {
			qc.Error = utils.StackError(err, "Failed to parse measure: %s", measure.Expr)
			return
		}
		measure.ExprParsed = expr.Rewrite(qc, measure.ExprParsed)
		if qc.Error != nil {
			return
		}

		measure.FiltersParsed = make([]expr.Expr, len(measure.Filters))
		for j, filter := range measure.Filters {
			measure.FiltersParsed[j], err = expr.ParseExpr(filter)
			if err != nil {
				qc.Error = utils.StackError(err, "Failed to parse measure filter %s", filter)
				return
			}
			measure.FiltersParsed[j] = expr.Rewrite(qc, measure.FiltersParsed[j])
			if qc.Error != nil {
				return
			}
		}
		measure.FiltersParsed = normalizeAndFilters(measure.FiltersParsed)
		qc.AQLQuery.Measures[i] = measure
	}

	// ony support 1 measure for now
	if len(qc.AQLQuery.Measures) != 1 {
		qc.Error = utils.StackError(nil, "expect one measure per query, but got %d",
			len(qc.AQLQuery.Measures))
		return
	}

	if _, ok := qc.AQLQuery.Measures[0].ExprParsed.(*expr.NumberLiteral); ok {
		qc.IsNonAggregationQuery = true
		// in case user forgot to provide limit
		if qc.AQLQuery.Limit == 0 {
			qc.AQLQuery.Limit = nonAggregationQueryLimit
		}
		return
	}

	aggregate, ok := qc.AQLQuery.Measures[0].ExprParsed.(*expr.Call)
	if !ok {
		qc.Error = utils.StackError(nil, "expect aggregate function, but got %s",
			qc.AQLQuery.Measures[0].Expr)
		return
	}

	if len(aggregate.Args) != 1 {
		qc.Error = utils.StackError(nil,
			"expect one parameter for aggregate function %s, but got %d",
			aggregate.Name, len(aggregate.Args))
		return
	}

	if qc.ReturnHLLBinary && aggregate.Name != expr.HllCallName {
		qc.Error = utils.StackError(nil, "expect hll aggregate function as client specify 'Accept' as "+
			"'application/hll', but got %s",
			qc.AQLQuery.Measures[0].Expr)
		return
	}
}

func (qc *QueryContext) processDimensions() {
	rawDims := qc.AQLQuery.Dimensions
	qc.AQLQuery.Dimensions = []common.Dimension{}
	qc.DimensionVectorIndex = make([]int, len(rawDims))
	for _, dim := range rawDims {
		var err error
		dim.ExprParsed, err = expr.ParseExpr(dim.Expr)
		if err != nil {
			qc.Error = utils.StackError(err, "Failed to parse dimension: %s", dim.Expr)
			return
		}
		if _, ok := dim.ExprParsed.(*expr.Wildcard); ok && qc.IsNonAggregationQuery {
			qc.AQLQuery.Dimensions = append(qc.AQLQuery.Dimensions, qc.getAllColumnsDimension()...)
		} else {
			qc.AQLQuery.Dimensions = append(qc.AQLQuery.Dimensions, dim)
		}
	}

	for idx, dim := range qc.AQLQuery.Dimensions {
		dim.ExprParsed = expr.Rewrite(qc, dim.ExprParsed)
		if vr, ok := dim.ExprParsed.(*expr.VarRef); ok {
			if len(vr.EnumReverseDict) > 0 {
				qc.DimensionEnumReverseDicts[idx] = vr.EnumReverseDict
			}
		}
		qc.AQLQuery.Dimensions[idx] = dim
	}
}

func (qc *QueryContext) sortDimensionColumns() {
	orderedIndex := 0
	numDimensions := len(qc.AQLQuery.Dimensions)
	qc.DimensionVectorIndex = make([]int, numDimensions)
	byteWidth := 1 << uint(len(qc.NumDimsPerDimWidth)-1)
	for byteIndex := range qc.NumDimsPerDimWidth {
		for originIndex, dim := range qc.AQLQuery.Dimensions {
			dataBytes := common.GetDimensionDataBytes(dim.ExprParsed)
			if dataBytes == byteWidth {
				// record value offset, null offset pair
				// null offsets will have to add total dim bytes later
				qc.DimensionVectorIndex[originIndex] = orderedIndex
				qc.NumDimsPerDimWidth[byteIndex]++
				qc.DimRowBytes += dataBytes
				orderedIndex++
			}
		}
		byteWidth >>= 1
	}
	// plus one byte per dimension column for validity
	qc.DimRowBytes += numDimensions
}

func (qc *QueryContext) getAllColumnsDimension() (columns []common.Dimension) {
	// only main table columns wildcard match supported
	for _, column := range qc.Tables[0].Schema.Columns {
		if !column.Deleted && column.Type != metaCom.GeoShape {
			columns = append(columns, common.Dimension{
				Expr:       column.Name,
				ExprParsed: &expr.VarRef{Val: column.Name},
			})
		}
	}
	return
}

// Rewrite walks the expresison AST and resolves data types bottom up.
// In addition it also translates enum strings and rewrites their predicates.
// TODO: remove dup in aql_compiler.go
func (qc *QueryContext) Rewrite(expression expr.Expr) expr.Expr {
	switch e := expression.(type) {
	case *expr.ParenExpr:
		// Strip parenthesis from the input
		return e.Expr
	case *expr.VarRef:
		tableID, columnID, err := qc.resolveColumn(e.Val)
		if err != nil {
			qc.Error = err
			return expression
		}
		column := qc.Tables[tableID].Schema.Columns[columnID]
		if column.Deleted {
			qc.Error = utils.StackError(nil, "column %s of table %s has been deleted",
				column.Name, qc.Tables[tableID].Schema.Name)
			return expression
		}
		dataType := qc.Tables[tableID].ValueTypeByColumn[columnID]
		e.ExprType = common.DataTypeToExprType[dataType]
		e.TableID = tableID
		e.ColumnID = columnID
		dict := qc.Tables[tableID].EnumDicts[column.Name]
		e.EnumDict = dict.Dict
		e.EnumReverseDict = dict.ReverseDict
		e.DataType = dataType
		e.IsHLLColumn = column.HLLConfig.IsHLLColumn
	case *expr.UnaryExpr:
		if expr.IsUUIDColumn(e.Expr) && e.Op != expr.GET_HLL_VALUE {
			qc.Error = utils.StackError(nil, "uuid column type only supports countdistincthll unary expression")
			return expression
		}

		if err := blockNumericOpsForColumnOverFourBytes(e.Op, e.Expr); err != nil {
			qc.Error = err
			return expression
		}

		e.ExprType = e.Expr.Type()
		switch e.Op {
		case expr.EXCLAMATION, expr.NOT, expr.IS_FALSE:
			e.ExprType = expr.Boolean
			// Normalize the operator.
			e.Op = expr.NOT
			e.Expr = expr.Cast(e.Expr, expr.Boolean)
			childExpr := e.Expr
			callRef, isCallRef := childExpr.(*expr.Call)
			if isCallRef && callRef.Name == expr.GeographyIntersectsCallName {
				qc.Error = utils.StackError(nil, "Not %s condition is not allowed", expr.GeographyIntersectsCallName)
				break
			}
		case expr.UNARY_MINUS:
			// Upgrade to signed.
			if e.ExprType < expr.Signed {
				e.ExprType = expr.Signed
			}
		case expr.IS_NULL, expr.IS_NOT_NULL:
			e.ExprType = expr.Boolean
		case expr.IS_TRUE:
			// Strip IS_TRUE if child is already boolean.
			if e.Expr.Type() == expr.Boolean {
				return e.Expr
			}
			// Rewrite to NOT(NOT(child)).
			e.ExprType = expr.Boolean
			e.Op = expr.NOT
			e.Expr = expr.Cast(e.Expr, expr.Boolean)
			return &expr.UnaryExpr{Expr: e, Op: expr.NOT, ExprType: expr.Boolean}
		case expr.BITWISE_NOT:
			// Cast child to unsigned.
			e.ExprType = expr.Unsigned
			e.Expr = expr.Cast(e.Expr, expr.Unsigned)
		case expr.GET_MONTH_START, expr.GET_QUARTER_START, expr.GET_YEAR_START, expr.GET_WEEK_START:
			// Cast child to unsigned.
			e.ExprType = expr.Unsigned
			e.Expr = expr.Cast(e.Expr, expr.Unsigned)
		case expr.GET_DAY_OF_MONTH, expr.GET_DAY_OF_YEAR, expr.GET_MONTH_OF_YEAR, expr.GET_QUARTER_OF_YEAR:
			// Cast child to unsigned.
			e.ExprType = expr.Unsigned
			e.Expr = expr.Cast(e.Expr, expr.Unsigned)
		case expr.GET_HLL_VALUE:
			e.ExprType = expr.Unsigned
			e.Expr = expr.Cast(e.Expr, expr.Unsigned)
		default:
			qc.Error = utils.StackError(nil, "unsupported unary expression %s",
				e.String())
		}
	case *expr.BinaryExpr:
		if err := blockNumericOpsForColumnOverFourBytes(e.Op, e.LHS, e.RHS); err != nil {
			qc.Error = err
			return expression
		}

		if e.Op != expr.EQ && e.Op != expr.NEQ {
			_, isRHSStr := e.RHS.(*expr.StringLiteral)
			_, isLHSStr := e.LHS.(*expr.StringLiteral)
			if isRHSStr || isLHSStr {
				qc.Error = utils.StackError(nil, "string type only support EQ and NEQ operators")
				return expression
			}
		}
		highestType := e.LHS.Type()
		if e.RHS.Type() > highestType {
			highestType = e.RHS.Type()
		}
		switch e.Op {
		case expr.ADD, expr.SUB:
			// Upgrade and cast to highestType.
			e.ExprType = highestType
			if highestType == expr.Float {
				e.LHS = expr.Cast(e.LHS, expr.Float)
				e.RHS = expr.Cast(e.RHS, expr.Float)
			} else if e.Op == expr.SUB {
				// For lhs - rhs, upgrade to signed at least.
				e.ExprType = expr.Signed
			}
		case expr.MUL, expr.MOD:
			// Upgrade and cast to highestType.
			e.ExprType = highestType
			e.LHS = expr.Cast(e.LHS, highestType)
			e.RHS = expr.Cast(e.RHS, highestType)
		case expr.DIV:
			// Upgrade and cast to float.
			e.ExprType = expr.Float
			e.LHS = expr.Cast(e.LHS, expr.Float)
			e.RHS = expr.Cast(e.RHS, expr.Float)
		case expr.BITWISE_AND, expr.BITWISE_OR, expr.BITWISE_XOR,
			expr.BITWISE_LEFT_SHIFT, expr.BITWISE_RIGHT_SHIFT, expr.FLOOR, expr.CONVERT_TZ:
			// Cast to unsigned.
			e.ExprType = expr.Unsigned
			e.LHS = expr.Cast(e.LHS, expr.Unsigned)
			e.RHS = expr.Cast(e.RHS, expr.Unsigned)
		case expr.AND, expr.OR:
			// Cast to boolean.
			e.ExprType = expr.Boolean
			e.LHS = expr.Cast(e.LHS, expr.Boolean)
			e.RHS = expr.Cast(e.RHS, expr.Boolean)
		case expr.LT, expr.LTE, expr.GT, expr.GTE:
			// Cast to boolean.
			e.ExprType = expr.Boolean
			e.LHS = expr.Cast(e.LHS, highestType)
			e.RHS = expr.Cast(e.RHS, highestType)
		case expr.NEQ, expr.EQ:
			// swap lhs and rhs if rhs is VarRef but lhs is not.
			if _, lhsVarRef := e.LHS.(*expr.VarRef); !lhsVarRef {
				if _, rhsVarRef := e.RHS.(*expr.VarRef); rhsVarRef {
					e.LHS, e.RHS = e.RHS, e.LHS
				}
			}

			e.ExprType = expr.Boolean
			// Match enum = 'case' and enum != 'case'.

			lhs, _ := e.LHS.(*expr.VarRef)
			// rhs is bool
			rhsBool, _ := e.RHS.(*expr.BooleanLiteral)
			if lhs != nil && rhsBool != nil {
				if (e.Op == expr.EQ && rhsBool.Val) || (e.Op == expr.NEQ && !rhsBool.Val) {
					return &expr.UnaryExpr{Expr: lhs, Op: expr.IS_TRUE, ExprType: expr.Boolean}
				}
				return &expr.UnaryExpr{Expr: lhs, Op: expr.NOT, ExprType: expr.Boolean}
			}

			// rhs is string enum
			rhs, _ := e.RHS.(*expr.StringLiteral)
			if lhs != nil && rhs != nil && lhs.EnumDict != nil {
				// Enum dictionary translation
				value, exists := lhs.EnumDict[rhs.Val]
				if !exists {
					// Combination of nullable data with not/and/or operators on top makes
					// short circuiting hard.
					// To play it safe we match against an invalid value.
					value = -1
				}
				e.RHS = &expr.NumberLiteral{Int: value, ExprType: expr.Unsigned}
				break
			}

			// Cast to highestType.
			e.LHS = expr.Cast(e.LHS, highestType)
			e.RHS = expr.Cast(e.RHS, highestType)

			if rhs != nil && lhs.DataType == memCom.GeoPoint {
				if val, err := memCom.GeoPointFromString(rhs.Val); err != nil {
					qc.Error = err
				} else {
					e.RHS = &expr.GeopointLiteral{
						Val: val,
					}
				}
			}
		case expr.IN:
			return qc.expandINop(e)
		case expr.NOT_IN:
			return &expr.UnaryExpr{
				Op:   expr.NOT,
				Expr: qc.expandINop(e),
			}
		default:
			qc.Error = utils.StackError(nil, "unsupported binary expression %s",
				e.String())
		}
	case *expr.Call:
		e.Name = strings.ToLower(e.Name)
		switch e.Name {
		case expr.ConvertTzCallName:
			if len(e.Args) != 3 {
				qc.Error = utils.StackError(
					nil, "convert_tz must have 3 arguments",
				)
				break
			}
			fromTzStringExpr, isStrLiteral := e.Args[1].(*expr.StringLiteral)
			if !isStrLiteral {
				qc.Error = utils.StackError(nil, "2nd argument of convert_tz must be a string")
				break
			}
			toTzStringExpr, isStrLiteral := e.Args[2].(*expr.StringLiteral)
			if !isStrLiteral {
				qc.Error = utils.StackError(nil, "3rd argument of convert_tz must be a string")
				break
			}
			fromTz, err := common.ParseTimezone(fromTzStringExpr.Val)
			if err != nil {
				qc.Error = utils.StackError(err, "failed to rewrite convert_tz")
				break
			}
			toTz, err := common.ParseTimezone(toTzStringExpr.Val)
			if err != nil {
				qc.Error = utils.StackError(err, "failed to rewrite convert_tz")
				break
			}
			_, fromOffsetInSeconds := utils.Now().In(fromTz).Zone()
			_, toOffsetInSeconds := utils.Now().In(toTz).Zone()
			offsetInSeconds := toOffsetInSeconds - fromOffsetInSeconds
			return &expr.BinaryExpr{
				Op:  expr.ADD,
				LHS: e.Args[0],
				RHS: &expr.NumberLiteral{
					Int:      offsetInSeconds,
					Expr:     strconv.Itoa(offsetInSeconds),
					ExprType: expr.Unsigned,
				},
				ExprType: expr.Unsigned,
			}
		case expr.CountCallName:
			e.ExprType = expr.Unsigned
		case expr.DayOfWeekCallName:
			// dayofweek from ts: (ts / secondsInDay + 4) % 7 + 1
			// ref: https://dev.mysql.com/doc/refman/5.5/en/date-and-time-functions.html#function_dayofweek
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(nil, "dayofweek takes exactly 1 argument")
				break
			}
			tsExpr := e.Args[0]
			return &expr.BinaryExpr{
				Op:       expr.ADD,
				ExprType: expr.Unsigned,
				RHS: &expr.NumberLiteral{
					Int:      1,
					Expr:     "1",
					ExprType: expr.Unsigned,
				},
				LHS: &expr.BinaryExpr{
					Op:       expr.MOD,
					ExprType: expr.Unsigned,
					RHS: &expr.NumberLiteral{
						Int:      common.DaysPerWeek,
						Expr:     strconv.Itoa(common.DaysPerWeek),
						ExprType: expr.Unsigned,
					},
					LHS: &expr.BinaryExpr{
						Op:       expr.ADD,
						ExprType: expr.Unsigned,
						RHS: &expr.NumberLiteral{
							// offset for
							Int:      common.WeekdayOffset,
							Expr:     strconv.Itoa(common.WeekdayOffset),
							ExprType: expr.Unsigned,
						},
						LHS: &expr.BinaryExpr{
							Op:       expr.DIV,
							ExprType: expr.Unsigned,
							RHS: &expr.NumberLiteral{
								Int:      common.SecondsPerDay,
								Expr:     strconv.Itoa(common.SecondsPerDay),
								ExprType: expr.Unsigned,
							},
							LHS: tsExpr,
						},
					},
				},
			}
			// no-op, this will be over written
		case expr.FromUnixTimeCallName:
			// for now, only the following format is allowed for backward compatibility
			// from_unixtime(time_col / 1000)
			timeColumnDivideErrMsg := "from_unixtime must be time column / 1000"
			timeColDivide, isBinary := e.Args[0].(*expr.BinaryExpr)
			if !isBinary || timeColDivide.Op != expr.DIV {
				qc.Error = utils.StackError(nil, timeColumnDivideErrMsg)
				break
			}
			divisor, isLiteral := timeColDivide.RHS.(*expr.NumberLiteral)
			if !isLiteral || divisor.Int != 1000 {
				qc.Error = utils.StackError(nil, timeColumnDivideErrMsg)
				break
			}
			if par, isParen := timeColDivide.LHS.(*expr.ParenExpr); isParen {
				timeColDivide.LHS = par.Expr
			}
			timeColExpr, isVarRef := timeColDivide.LHS.(*expr.VarRef)
			if !isVarRef {
				qc.Error = utils.StackError(nil, timeColumnDivideErrMsg)
				break
			}
			return timeColExpr
		case expr.HourCallName:
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(nil, "hour takes exactly 1 argument")
				break
			}
			// hour(ts) = (ts % secondsInDay) / secondsInHour
			return &expr.BinaryExpr{
				Op:       expr.DIV,
				ExprType: expr.Unsigned,
				LHS: &expr.BinaryExpr{
					Op:  expr.MOD,
					LHS: e.Args[0],
					RHS: &expr.NumberLiteral{
						Expr:     strconv.Itoa(common.SecondsPerDay),
						Int:      common.SecondsPerDay,
						ExprType: expr.Unsigned,
					},
				},
				RHS: &expr.NumberLiteral{
					Expr:     strconv.Itoa(common.SecondsPerHour),
					Int:      common.SecondsPerHour,
					ExprType: expr.Unsigned,
				},
			}
			// list of literals, no need to cast it for now.
		case expr.ListCallName:
		case expr.GeographyIntersectsCallName:
			if len(e.Args) != 2 {
				qc.Error = utils.StackError(
					nil, "expect 2 argument for %s, but got %s", e.Name, e.String())
				break
			}

			lhsRef, isVarRef := e.Args[0].(*expr.VarRef)
			if !isVarRef || (lhsRef.DataType != memCom.GeoShape && lhsRef.DataType != memCom.GeoPoint) {
				qc.Error = utils.StackError(
					nil, "expect argument to be a valid geo shape or geo point column for %s, but got %s of type %s",
					e.Name, e.Args[0].String(), memCom.DataTypeName[lhsRef.DataType])
				break
			}

			lhsGeoPoint := lhsRef.DataType == memCom.GeoPoint

			rhsRef, isVarRef := e.Args[1].(*expr.VarRef)
			if !isVarRef || (rhsRef.DataType != memCom.GeoShape && rhsRef.DataType != memCom.GeoPoint) {
				qc.Error = utils.StackError(
					nil, "expect argument to be a valid geo shape or geo point column for %s, but got %s of type %s",
					e.Name, e.Args[1].String(), memCom.DataTypeName[rhsRef.DataType])
				break
			}

			rhsGeoPoint := rhsRef.DataType == memCom.GeoPoint

			if lhsGeoPoint == rhsGeoPoint {
				qc.Error = utils.StackError(
					nil, "expect exactly one geo shape column and one geo point column for %s, got %s",
					e.Name, e.String())
				break
			}

			// Switch geo point so that lhs is geo shape and rhs is geo point
			if lhsGeoPoint {
				e.Args[0], e.Args[1] = e.Args[1], e.Args[0]
			}

			e.ExprType = expr.Boolean
		case expr.HexCallName:
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(
					nil, "expect 1 argument for %s, but got %s", e.Name, e.String())
				break
			}
			colRef, isVarRef := e.Args[0].(*expr.VarRef)
			if !isVarRef || colRef.DataType != memCom.UUID {
				qc.Error = utils.StackError(
					nil, "expect 1 argument to be a valid uuid column for %s, but got %s of type %s",
					e.Name, e.Args[0].String(), memCom.DataTypeName[colRef.DataType])
				break
			}
			e.ExprType = e.Args[0].Type()
		case expr.CountDistinctHllCallName:
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(
					nil, "expect 1 argument for %s, but got %s", e.Name, e.String())
				break
			}
			colRef, isVarRef := e.Args[0].(*expr.VarRef)
			if !isVarRef {
				qc.Error = utils.StackError(
					nil, "expect 1 argument to be a column for %s", e.Name)
				break
			}

			e.Name = expr.HllCallName
			// 1. noop when column itself is hll column
			// 2. compute hll on the fly when column is not hll column
			if !colRef.IsHLLColumn {
				e.Args[0] = &expr.UnaryExpr{
					Op:       expr.GET_HLL_VALUE,
					Expr:     colRef,
					ExprType: expr.Unsigned,
				}
			}
			e.ExprType = expr.Unsigned
		case expr.HllCallName:
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(
					nil, "expect 1 argument for %s, but got %s", e.Name, e.String())
				break
			}
			colRef, isVarRef := e.Args[0].(*expr.VarRef)
			if !isVarRef || colRef.DataType != memCom.Uint32 {
				qc.Error = utils.StackError(
					nil, "expect 1 argument to be a valid hll column for %s, but got %s of type %s",
					e.Name, e.Args[0].String(), memCom.DataTypeName[colRef.DataType])
				break
			}
			e.ExprType = e.Args[0].Type()
		case expr.SumCallName, expr.MinCallName, expr.MaxCallName, expr.AvgCallName:
			if len(e.Args) != 1 {
				qc.Error = utils.StackError(
					nil, "expect 1 argument for %s, but got %s", e.Name, e.String())
				break
			}
			// For avg, the expression type should always be float.
			if e.Name == expr.AvgCallName {
				e.Args[0] = expr.Cast(e.Args[0], expr.Float)
			}
			e.ExprType = e.Args[0].Type()
		case expr.LengthCallName, expr.ContainsCallName, expr.ElementAtCallName:
			// validate first argument
			if len(e.Args) == 0 {
				qc.Error = utils.StackError(
					nil, "array function %s requires arguments", e.Name)
				break
			}
			firstArg := e.Args[0]
			vr, ok := firstArg.(*expr.VarRef)
			if !ok || !memCom.IsArrayType(vr.DataType) {
				qc.Error = utils.StackError(
					nil, "array function %s requires first argument to be array type column, but got %s", e.Name, firstArg)
			}

			if e.Name == expr.LengthCallName {
				if len(e.Args) != 1 {
					qc.Error = utils.StackError(
						nil, "array function %s takes exactly 1 argument", e.Name)
					break
				}
				e.ExprType = expr.Unsigned
			} else if e.Name == expr.ContainsCallName {
				if len(e.Args) != 2 {
					qc.Error = utils.StackError(
						nil, "array function %s takes exactly 2 arguments", e.Name)
					break
				}
				e.ExprType = expr.Boolean
				// we don't do type checks at broker
			} else if e.Name == expr.ElementAtCallName {
				if len(e.Args) != 2 {
					qc.Error = utils.StackError(
						nil, "array function %s takes exactly 2 arguments", e.Name)
					break
				}
				if _, ok := e.Args[1].(*expr.NumberLiteral); !ok {
					qc.Error = utils.StackError(
						nil, "array function %s takes array type column and an index", e.Name)
				}
				e.ExprType = vr.ExprType
			}

		default:
			qc.Error = utils.StackError(nil, "unknown function %s", e.Name)
		}
	case *expr.Case:
		highestType := e.Else.Type()
		for _, whenThen := range e.WhenThens {
			if whenThen.Then.Type() > highestType {
				highestType = whenThen.Then.Type()
			}
		}
		// Cast else and thens to highestType, cast whens to boolean.
		e.Else = expr.Cast(e.Else, highestType)
		for i, whenThen := range e.WhenThens {
			whenThen.When = expr.Cast(whenThen.When, expr.Boolean)
			whenThen.Then = expr.Cast(whenThen.Then, highestType)
			e.WhenThens[i] = whenThen
		}
		e.ExprType = highestType
	}
	return expression
}

// TODO: remove dup in aql_compiler.go
func blockNumericOpsForColumnOverFourBytes(token expr.Token, expressions ...expr.Expr) error {
	if token == expr.UNARY_MINUS || token == expr.BITWISE_NOT ||
		(token >= expr.ADD && token <= expr.BITWISE_LEFT_SHIFT) {
		for _, expression := range expressions {
			if varRef, isVarRef := expression.(*expr.VarRef); isVarRef && memCom.DataTypeBytes(varRef.DataType) > 4 {
				return utils.StackError(nil, "numeric operations not supported for column over 4 bytes length, got %s", expression.String())
			}
		}
	}
	return nil
}

func (qc *QueryContext) expandINop(e *expr.BinaryExpr) (expandedExpr expr.Expr) {
	lhs, ok := e.LHS.(*expr.VarRef)
	if !ok {
		qc.Error = utils.StackError(nil, "lhs of IN or NOT_IN must be a valid column")
	}
	rhs := e.RHS
	switch rhsTyped := rhs.(type) {
	case *expr.Call:
		expandedExpr = &expr.BooleanLiteral{Val: false}
		for _, value := range rhsTyped.Args {
			switch expandedExpr.(type) {
			case *expr.BooleanLiteral:
				expandedExpr = qc.Rewrite(&expr.BinaryExpr{
					Op:  expr.EQ,
					LHS: lhs,
					RHS: value,
				}).(*expr.BinaryExpr)
			default:
				lastExpr := expandedExpr
				expandedExpr = &expr.BinaryExpr{
					Op:  expr.OR,
					LHS: lastExpr,
					RHS: qc.Rewrite(&expr.BinaryExpr{
						Op:  expr.EQ,
						LHS: lhs,
						RHS: value,
					}).(*expr.BinaryExpr),
				}
			}
		}
		break
	default:
		qc.Error = utils.StackError(nil, "only EQ and IN operators are supported for geo fields")
	}
	return
}

// TODO remove duplicate in aql_compiler.go
// normalizeAndFilters extracts top AND operators and flatten them out to the
// filter slice.
func normalizeAndFilters(filters []expr.Expr) []expr.Expr {
	i := 0
	for i < len(filters) {
		f, _ := filters[i].(*expr.BinaryExpr)
		if f != nil && f.Op == expr.AND {
			filters[i] = f.LHS
			filters = append(filters, f.RHS)
		} else {
			i++
		}
	}
	return filters
}