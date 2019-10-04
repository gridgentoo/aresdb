// Code generated from query/sql/SqlBase.g4 by ANTLR 4.7.1. DO NOT EDIT.

package antlrgen // SqlBase
import "github.com/antlr/antlr4/runtime/Go/antlr"

// A complete Visitor for a parse tree produced by SqlBaseParser.
type SqlBaseVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by SqlBaseParser#statementDefault.
	VisitStatementDefault(ctx *StatementDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#query.
	VisitQuery(ctx *QueryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#with.
	VisitWith(ctx *WithContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#queryNoWith.
	VisitQueryNoWith(ctx *QueryNoWithContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#queryTermDefault.
	VisitQueryTermDefault(ctx *QueryTermDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#setOperation.
	VisitSetOperation(ctx *SetOperationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#queryPrimaryDefault.
	VisitQueryPrimaryDefault(ctx *QueryPrimaryDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#table.
	VisitTable(ctx *TableContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#inlineTable.
	VisitInlineTable(ctx *InlineTableContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#subquery.
	VisitSubquery(ctx *SubqueryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#sortItem.
	VisitSortItem(ctx *SortItemContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#querySpecification.
	VisitQuerySpecification(ctx *QuerySpecificationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#groupBy.
	VisitGroupBy(ctx *GroupByContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#singleGroupingSet.
	VisitSingleGroupingSet(ctx *SingleGroupingSetContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#groupingExpressions.
	VisitGroupingExpressions(ctx *GroupingExpressionsContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#namedQuery.
	VisitNamedQuery(ctx *NamedQueryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#setQuantifier.
	VisitSetQuantifier(ctx *SetQuantifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#selectSingle.
	VisitSelectSingle(ctx *SelectSingleContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#selectAll.
	VisitSelectAll(ctx *SelectAllContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#relationDefault.
	VisitRelationDefault(ctx *RelationDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#joinRelation.
	VisitJoinRelation(ctx *JoinRelationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#joinType.
	VisitJoinType(ctx *JoinTypeContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#joinCriteria.
	VisitJoinCriteria(ctx *JoinCriteriaContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#sampledRelation.
	VisitSampledRelation(ctx *SampledRelationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#sampleType.
	VisitSampleType(ctx *SampleTypeContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#aliasedRelation.
	VisitAliasedRelation(ctx *AliasedRelationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#columnAliases.
	VisitColumnAliases(ctx *ColumnAliasesContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#tableName.
	VisitTableName(ctx *TableNameContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#subqueryRelation.
	VisitSubqueryRelation(ctx *SubqueryRelationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#parenthesizedRelation.
	VisitParenthesizedRelation(ctx *ParenthesizedRelationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#logicalNot.
	VisitLogicalNot(ctx *LogicalNotContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#booleanDefault.
	VisitBooleanDefault(ctx *BooleanDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#logicalBinary.
	VisitLogicalBinary(ctx *LogicalBinaryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#predicated.
	VisitPredicated(ctx *PredicatedContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#comparison.
	VisitComparison(ctx *ComparisonContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#quantifiedComparison.
	VisitQuantifiedComparison(ctx *QuantifiedComparisonContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#between.
	VisitBetween(ctx *BetweenContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#inList.
	VisitInList(ctx *InListContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#inSubquery.
	VisitInSubquery(ctx *InSubqueryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#valueExpressionDefault.
	VisitValueExpressionDefault(ctx *ValueExpressionDefaultContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#concatenation.
	VisitConcatenation(ctx *ConcatenationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#arithmeticBinary.
	VisitArithmeticBinary(ctx *ArithmeticBinaryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#arithmeticUnary.
	VisitArithmeticUnary(ctx *ArithmeticUnaryContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#atTimeZone.
	VisitAtTimeZone(ctx *AtTimeZoneContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#dereference.
	VisitDereference(ctx *DereferenceContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#columnReference.
	VisitColumnReference(ctx *ColumnReferenceContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#nullLiteral.
	VisitNullLiteral(ctx *NullLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#rowConstructor.
	VisitRowConstructor(ctx *RowConstructorContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#subscript.
	VisitSubscript(ctx *SubscriptContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#typeConstructor.
	VisitTypeConstructor(ctx *TypeConstructorContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#specialDateTimeFunction.
	VisitSpecialDateTimeFunction(ctx *SpecialDateTimeFunctionContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#subqueryExpression.
	VisitSubqueryExpression(ctx *SubqueryExpressionContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#binaryLiteral.
	VisitBinaryLiteral(ctx *BinaryLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#currentUser.
	VisitCurrentUser(ctx *CurrentUserContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#parenthesizedExpression.
	VisitParenthesizedExpression(ctx *ParenthesizedExpressionContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#stringLiteral.
	VisitStringLiteral(ctx *StringLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#arrayConstructor.
	VisitArrayConstructor(ctx *ArrayConstructorContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#functionCall.
	VisitFunctionCall(ctx *FunctionCallContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#intervalLiteral.
	VisitIntervalLiteral(ctx *IntervalLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#numericLiteral.
	VisitNumericLiteral(ctx *NumericLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#booleanLiteral.
	VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#groupingOperation.
	VisitGroupingOperation(ctx *GroupingOperationContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#basicStringLiteral.
	VisitBasicStringLiteral(ctx *BasicStringLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#unicodeStringLiteral.
	VisitUnicodeStringLiteral(ctx *UnicodeStringLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#timeZoneInterval.
	VisitTimeZoneInterval(ctx *TimeZoneIntervalContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#timeZoneString.
	VisitTimeZoneString(ctx *TimeZoneStringContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#comparisonOperator.
	VisitComparisonOperator(ctx *ComparisonOperatorContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#comparisonQuantifier.
	VisitComparisonQuantifier(ctx *ComparisonQuantifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#booleanValue.
	VisitBooleanValue(ctx *BooleanValueContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#interval.
	VisitInterval(ctx *IntervalContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#intervalField.
	VisitIntervalField(ctx *IntervalFieldContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#normalForm.
	VisitNormalForm(ctx *NormalFormContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#sqltype.
	VisitSqltype(ctx *SqltypeContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#typeParameter.
	VisitTypeParameter(ctx *TypeParameterContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#baseType.
	VisitBaseType(ctx *BaseTypeContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#whenClause.
	VisitWhenClause(ctx *WhenClauseContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#filter.
	VisitFilter(ctx *FilterContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#qualifiedName.
	VisitQualifiedName(ctx *QualifiedNameContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#unquotedIdentifier.
	VisitUnquotedIdentifier(ctx *UnquotedIdentifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#quotedIdentifier.
	VisitQuotedIdentifier(ctx *QuotedIdentifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#backQuotedIdentifier.
	VisitBackQuotedIdentifier(ctx *BackQuotedIdentifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#digitIdentifier.
	VisitDigitIdentifier(ctx *DigitIdentifierContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#decimalLiteral.
	VisitDecimalLiteral(ctx *DecimalLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#doubleLiteral.
	VisitDoubleLiteral(ctx *DoubleLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#integerLiteral.
	VisitIntegerLiteral(ctx *IntegerLiteralContext) interface{}

	// Visit a parse tree produced by SqlBaseParser#nonReserved.
	VisitNonReserved(ctx *NonReservedContext) interface{}
}
