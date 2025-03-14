package util

import java.lang.annotation.Documented
import java.lang.annotation.ElementType
import java.lang.annotation.Retention
import java.lang.annotation.RetentionPolicy
import java.lang.annotation.Target

import groovy.transform.CompileStatic
import org.codehaus.groovy.ast.ASTNode
import org.codehaus.groovy.ast.AnnotationNode
import org.codehaus.groovy.ast.ClassHelper
import org.codehaus.groovy.ast.MethodNode
import org.codehaus.groovy.ast.Parameter
import org.codehaus.groovy.ast.VariableScope
import org.codehaus.groovy.ast.expr.ArgumentListExpression
import org.codehaus.groovy.ast.expr.ClosureExpression
import org.codehaus.groovy.ast.expr.ConstantExpression
import org.codehaus.groovy.ast.expr.MethodCallExpression
import org.codehaus.groovy.ast.expr.StaticMethodCallExpression
import org.codehaus.groovy.ast.expr.VariableExpression
import org.codehaus.groovy.ast.stmt.ReturnStatement
import org.codehaus.groovy.control.SourceUnit
import org.codehaus.groovy.transform.AbstractASTTransformation
import org.codehaus.groovy.transform.GroovyASTTransformation
import org.codehaus.groovy.transform.GroovyASTTransformationClass

/**
 * Retry the method in case of exception. Using Helpers.evaluateWithRetry
 */
@Documented
@Retention(RetentionPolicy.RUNTIME)
@Target(ElementType.METHOD)
@GroovyASTTransformationClass(classes = RetryASTTransformation)
@interface Retry {

    /**
     * How many times to retry.
     * @return Number of attempts
     */
    int attempts() default 3

    /**
     * Delay in seconds between attempts, in time units.
     * @return Delay
     */
    long delay() default 1
}

@GroovyASTTransformation
class RetryASTTransformation extends AbstractASTTransformation {

    @Override
    @CompileStatic
    void visit(ASTNode[] nodes, SourceUnit sourceUnit) {
        AnnotationNode annotation = (AnnotationNode) nodes[0]
        MethodNode method = (MethodNode) nodes[1]

        def clazz = method.declaringClass
        def methodName = method.name + "_with_retry"
        clazz.addMethod(
                methodName,
                method.modifiers,
                method.returnType,
                method.parameters,
                method.exceptions,
                method.code
        )

        int attempts = getMemberIntValue(annotation, "attempts")
        int delay = getMemberIntValue(annotation, "delay")

        def argumentListExpression = new ArgumentListExpression(method.parameters)
        def funcCall = new MethodCallExpression(new VariableExpression("this"), methodName, argumentListExpression)

        def closureExpression = new ClosureExpression(Parameter.EMPTY_ARRAY, new ReturnStatement(funcCall))
        def variableScope = new VariableScope()
        method.parameters.each {
            variableScope.putReferencedLocalVariable(it)
            it.setClosureSharedVariable(true)
        }
        closureExpression.setVariableScope(variableScope)

        def retryCall = new ReturnStatement(
                new StaticMethodCallExpression(
                        ClassHelper.make(Helpers),
                        "evaluateWithRetry",
                        new ArgumentListExpression(
                                new ConstantExpression(attempts),
                                new ConstantExpression(delay),
                                closureExpression
                        )
                ))

        method.setCode(retryCall)
    }
}
