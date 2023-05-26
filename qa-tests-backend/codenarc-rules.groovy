ruleset {

    description '''
        A Sample Groovy RuleSet containing all CodeNarc Rules, grouped by category.
        You can use this as a template for your own custom RuleSet.
        Just delete the rules that you don't want to include.
        '''

    // rulesets/basic.xml
    AssertWithinFinallyBlock 
    AssignmentInConditional 
    BigDecimalInstantiation 
    BitwiseOperatorInConditional 
    BooleanGetBoolean 
    BrokenNullCheck 
    BrokenOddnessCheck 
    ClassForName 
    ComparisonOfTwoConstants 
    ComparisonWithSelf 
    ConstantAssertExpression 
    ConstantIfExpression 
    ConstantTernaryExpression 
    DeadCode 
    DoubleNegative 
    DuplicateCaseStatement 
    DuplicateMapKey 
    DuplicateSetValue 
    EmptyCatchBlock 
    // EmptyClass
    EmptyElseBlock 
    EmptyFinallyBlock 
    EmptyForStatement 
    EmptyIfStatement 
    EmptyInstanceInitializer 
    // EmptyMethod
    EmptyStaticInitializer 
    EmptySwitchStatement 
    EmptySynchronizedStatement 
    EmptyTryBlock 
    EmptyWhileStatement 
    EqualsAndHashCode 
    EqualsOverloaded 
    ExplicitGarbageCollection 
    ForLoopShouldBeWhileLoop 
    HardCodedWindowsFileSeparator 
    HardCodedWindowsRootDirectory
    IntegerGetInteger 
    MultipleUnaryOperators 
    RandomDoubleCoercedToZero 
    RemoveAllOnSelf 
    ReturnFromFinallyBlock 
    ThrowExceptionFromFinallyBlock 
    
    // rulesets/braces.xml
    ElseBlockBraces 
    ForStatementBraces 
    IfStatementBraces 
    WhileStatementBraces 
    
    // rulesets/concurrency.xml
    BusyWait 
    DoubleCheckedLocking 
    InconsistentPropertyLocking 
    InconsistentPropertySynchronization 
    NestedSynchronization 
    StaticCalendarField 
    StaticConnection 
    StaticDateFormatField 
    StaticMatcherField 
    StaticSimpleDateFormatField 
    SynchronizedMethod 
    SynchronizedOnBoxedPrimitive 
    SynchronizedOnGetClass 
    SynchronizedOnReentrantLock 
    SynchronizedOnString 
    SynchronizedOnThis 
    SynchronizedReadObjectMethod 
    SystemRunFinalizersOnExit 
    ThisReferenceEscapesConstructor 
    ThreadGroup 
    ThreadLocalNotStaticFinal 
    ThreadYield 
    UseOfNotifyMethod 
    VolatileArrayField 
    VolatileLongOrDoubleField 
    WaitOutsideOfWhileLoop 
    
    // rulesets/convention.xml
    ConfusingTernary 
    CouldBeElvis 
    // CouldBeSwitchStatement
    // FieldTypeRequired
    HashtableIsObsolete 
    IfStatementCouldBeTernary 
    InvertedCondition 
    InvertedIfElse 
    LongLiteralWithLowerCaseL 
    // MethodParameterTypeRequired
    // MethodReturnTypeRequired
    // NoDef
    NoTabCharacter 
    ParameterReassignment 
    TernaryCouldBeElvis 
    TrailingComma 
    // VariableTypeRequired
    VectorIsObsolete 
    
    // rulesets/design.xml
    AbstractClassWithPublicConstructor 
    AbstractClassWithoutAbstractMethod 
    AssignmentToStaticFieldFromInstanceMethod 
    BooleanMethodReturnsNull 
    // BuilderMethodWithSideEffects
    CloneableWithoutClone 
    CloseWithoutCloseable 
    CompareToWithoutComparable 
    ConstantsOnlyInterface 
    EmptyMethodInAbstractClass 
    FinalClassWithProtectedMember 
    ImplementationAsType 
    // Instanceof
    LocaleSetDefault 
    // NestedForLoop
    PrivateFieldCouldBeFinal 
    PublicInstanceField 
    ReturnsNullInsteadOfEmptyArray 
    ReturnsNullInsteadOfEmptyCollection 
    SimpleDateFormatMissingLocale 
    StatelessSingleton 
    ToStringReturnsNull 
    
    // rulesets/dry.xml
    // DuplicateListLiteral
    // DuplicateMapLiteral
    // DuplicateNumberLiteral
    // DuplicateStringLiteral
    
    // rulesets/enhanced.xml
    // CloneWithoutCloneable
    // JUnitAssertEqualsConstantActualValue
    // MissingOverrideAnnotation
    // UnsafeImplementationAsMap
    
    // rulesets/exceptions.xml
    CatchArrayIndexOutOfBoundsException 
    CatchError 
    // CatchException
    CatchIllegalMonitorStateException 
    CatchIndexOutOfBoundsException
//    CatchNullPointerException
    CatchRuntimeException 
    CatchThrowable 
    ConfusingClassNamedException 
    ExceptionExtendsError 
    ExceptionExtendsThrowable 
    ExceptionNotThrown 
    MissingNewInThrowStatement 
    ReturnNullFromCatchBlock 
    SwallowThreadDeath 
    ThrowError 
    ThrowException 
    ThrowNullPointerException 
    // ThrowRuntimeException
    ThrowThrowable 
    
    // rulesets/formatting.xml
    BlankLineBeforePackage 
    BlockEndsWithBlankLine 
    BlockStartsWithBlankLine 
    BracesForClass 
    BracesForForLoop 
    BracesForIfElse 
    BracesForMethod 
    BracesForTryCatchFinally 
    // ClassJavadoc
    // ClosureStatementOnOpeningLineOfMultipleLineClosure
    ConsecutiveBlankLines 
    FileEndsWithoutNewline
    Indentation

    LineLength 
    MissingBlankLineAfterImports 
    MissingBlankLineAfterPackage 
    SpaceAfterCatch 
    SpaceAfterClosingBrace 
    SpaceAfterComma 
    SpaceAfterFor 
    SpaceAfterIf 
    SpaceAfterOpeningBrace 
    SpaceAfterSemicolon 
    SpaceAfterSwitch 
    SpaceAfterWhile 
    SpaceAroundClosureArrow 
    // SpaceAroundMapEntryColon
    // SpaceAroundOperator
    SpaceBeforeClosingBrace 
    SpaceBeforeOpeningBrace 
    TrailingWhitespace 
    
    // rulesets/generic.xml
    IllegalClassMember 
    IllegalClassReference 
    IllegalPackageReference 
    IllegalRegex 
    IllegalString 
    IllegalSubclass 
    RequiredRegex 
    RequiredString 
    StatelessClass 
    
    // rulesets/grails.xml
    GrailsDomainHasEquals 
    GrailsDomainHasToString 
    GrailsDomainReservedSqlKeywordName 
    GrailsDomainWithServiceReference 
    GrailsDuplicateConstraint 
    GrailsDuplicateMapping 
    GrailsMassAssignment 
    GrailsPublicControllerMethod 
    GrailsServletContextReference 
    GrailsStatelessService 
    
    // rulesets/groovyism.xml
    AssignCollectionSort 
    AssignCollectionUnique 
    ClosureAsLastMethodParameter 
    CollectAllIsDeprecated 
    ConfusingMultipleReturns 
    // ExplicitArrayListInstantiation
    ExplicitCallToAndMethod 
    ExplicitCallToCompareToMethod 
    ExplicitCallToDivMethod 
    // ExplicitCallToEqualsMethod
    ExplicitCallToGetAtMethod 
    ExplicitCallToLeftShiftMethod 
    ExplicitCallToMinusMethod 
    ExplicitCallToModMethod 
    ExplicitCallToMultiplyMethod 
    ExplicitCallToOrMethod 
    ExplicitCallToPlusMethod 
    ExplicitCallToPowerMethod 
    ExplicitCallToRightShiftMethod 
    ExplicitCallToXorMethod 
    // ExplicitHashMapInstantiation
    ExplicitHashSetInstantiation 
    ExplicitLinkedHashMapInstantiation 
    // ExplicitLinkedListInstantiation
    ExplicitStackInstantiation 
    ExplicitTreeSetInstantiation 
    GStringAsMapKey 
    GStringExpressionWithinString 
    // GetterMethodCouldBeProperty
    GroovyLangImmutable 
    UseCollectMany 
    UseCollectNested 
    
    // rulesets/imports.xml
    DuplicateImport 
    ImportFromSamePackage 
    ImportFromSunPackages 
    MisorderedStaticImports 
    NoWildcardImports 
    UnnecessaryGroovyImport 
    UnusedImport 
    
    // rulesets/jdbc.xml
    DirectConnectionManagement 
    JdbcConnectionReference 
    JdbcResultSetReference 
    JdbcStatementReference 
    
    // rulesets/junit.xml
    ChainedTest 
    CoupledTestCase 
    JUnitAssertAlwaysFails 
    JUnitAssertAlwaysSucceeds 
    JUnitFailWithoutMessage 
    JUnitLostTest 
    JUnitPublicField 
    // JUnitPublicNonTestMethod
    JUnitPublicProperty 
    JUnitSetUpCallsSuper 
    JUnitStyleAssertions 
    JUnitTearDownCallsSuper 
    JUnitTestMethodWithoutAssert 
    JUnitUnnecessarySetUp 
    JUnitUnnecessaryTearDown 
    JUnitUnnecessaryThrowsException 
    SpockIgnoreRestUsed 
    UnnecessaryFail 
    UseAssertEqualsInsteadOfAssertTrue 
    UseAssertFalseInsteadOfNegation 
    UseAssertNullInsteadOfAssertEquals 
    UseAssertSameInsteadOfAssertTrue 
    UseAssertTrueInsteadOfAssertEquals 
    UseAssertTrueInsteadOfNegation 
    
    // rulesets/logging.xml
    // LoggerForDifferentClass
    // LoggerWithWrongModifiers
    LoggingSwallowsStacktrace
    // MultipleLoggers
    PrintStackTrace
    Println
    SystemErrPrint
    SystemOutPrint
    
    // rulesets/naming.xml
    AbstractClassName 
    ClassName 
    ClassNameSameAsFilename 
    ClassNameSameAsSuperclass 
    ConfusingMethodName 
    // FactoryMethodName
    FieldName 
    InterfaceName 
    InterfaceNameSameAsSuperInterface 
    // MethodName
    ObjectOverrideMisspelledMethodName 
    PackageName {
        regex = '[a-zA-Z]+[a-zA-Z0-9](.[a-zA-Z0-9]+)'
    }
    PackageNameMatchesFilePath 
    ParameterName 
    //PropertyName
    VariableName 
    
    // rulesets/security.xml
    // FileCreateTempFile
    InsecureRandom 
    // JavaIoPackageAccess
    // NonFinalPublicField
    NonFinalSubclassOfSensitiveInterface 
    ObjectFinalize 
    PublicFinalizeMethod 
    SystemExit 
    UnsafeArrayDeclaration 
    
    // rulesets/serialization.xml
    EnumCustomSerializationIgnored 
    SerialPersistentFields 
    SerialVersionUID 
    SerializableClassMustDefineSerialVersionUID 
    
    // rulesets/size.xml
    // AbcMetric   // Requires the GMetrics jar
    // ClassSize
    // CrapMetric   // Requires the GMetrics jar and a Cobertura coverage file
    // CyclomaticComplexity   // Requires the GMetrics jar
    // MethodCount
    // MethodSize
    NestedBlockDepth
    // ParameterCount
    
    // rulesets/unnecessary.xml
    AddEmptyString 
    ConsecutiveLiteralAppends 
    ConsecutiveStringConcatenation 
    UnnecessaryBigDecimalInstantiation 
    UnnecessaryBigIntegerInstantiation 
    UnnecessaryBooleanExpression 
    UnnecessaryBooleanInstantiation 
    UnnecessaryCallForLastElement 
    UnnecessaryCallToSubstring 
    UnnecessaryCast 
    UnnecessaryCatchBlock 
    UnnecessaryCollectCall 
    UnnecessaryCollectionCall 
    UnnecessaryConstructor 
    UnnecessaryDefInFieldDeclaration 
    UnnecessaryDefInMethodDeclaration 
    UnnecessaryDefInVariableDeclaration 
    UnnecessaryDotClass 
    UnnecessaryDoubleInstantiation 
    UnnecessaryElseStatement 
    UnnecessaryFinalOnPrivateMethod 
    UnnecessaryFloatInstantiation 
    // UnnecessaryGString
    // UnnecessaryGetter
    UnnecessaryIfStatement 
    UnnecessaryInstanceOfCheck 
    UnnecessaryInstantiationToGetClass 
    UnnecessaryIntegerInstantiation 
    UnnecessaryLongInstantiation 
    UnnecessaryModOne 
    UnnecessaryNullCheck 
    UnnecessaryNullCheckBeforeInstanceOf 
    //UnnecessaryObjectReferences
    UnnecessaryOverridingMethod 
    UnnecessaryPackageReference 
    UnnecessaryParenthesesForMethodCallWithClosure 
    UnnecessaryPublicModifier 
    // UnnecessaryReturnKeyword
    UnnecessarySafeNavigationOperator 
    UnnecessarySelfAssignment 
    UnnecessarySemicolon 
    // UnnecessarySetter
    UnnecessaryStringInstantiation 
    UnnecessaryTernaryExpression
    UnnecessaryToString 
    UnnecessaryTransientModifier 
    
    // rulesets/unused.xml
    UnusedArray 
    // UnusedMethodParameter
    UnusedObject 
    UnusedPrivateField 
    UnusedPrivateMethod 
    UnusedPrivateMethodParameter 
    UnusedVariable 
}
