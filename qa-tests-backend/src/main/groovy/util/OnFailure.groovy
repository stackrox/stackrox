package util

import org.spockframework.runtime.model.FieldInfo
import org.spockframework.runtime.model.MethodInfo
import org.spockframework.runtime.extension.ExtensionAnnotation
import org.spockframework.runtime.extension.IAnnotationDrivenExtension
import org.spockframework.runtime.extension.IMethodInterceptor
import org.spockframework.runtime.extension.IMethodInvocation
import org.spockframework.runtime.model.SpecInfo
import org.spockframework.runtime.model.FeatureInfo
import java.lang.annotation.ElementType
import java.lang.annotation.Retention
import java.lang.annotation.RetentionPolicy
import java.lang.annotation.Target

/**
 * Executes a handler when a test throws at any point.
 */
@Retention(RetentionPolicy.RUNTIME)
@Target(ElementType.TYPE)
@ExtensionAnnotation(OnFailureExtension)
@interface OnFailure {
    /**
     * Handler that is executed on failure.
     */
    Class<? extends Closure<?>> handler()
}

class OnFailureExtension implements IAnnotationDrivenExtension<OnFailure> {
    @Override
    void visitSpecAnnotation(OnFailure annotation, SpecInfo spec) {
        OnFailureInterceptor onFailureInterceptor = new OnFailureInterceptor(annotation)
        spec.getBottomSpec().getAllFixtureMethods().forEach {
            method ->
            method.addInterceptor(onFailureInterceptor)
        }
        spec.getBottomSpec().getAllFeatures().forEach {
            feature ->
            feature.getFeatureMethod().addInterceptor(onFailureInterceptor)
            feature.addIterationInterceptor(onFailureInterceptor)
        }
    }

    @Override
    void visitFeatureAnnotation(OnFailure annotation, FeatureInfo feature) {
    }

    @Override
    void visitFixtureAnnotation(OnFailure annotation, MethodInfo fixtureMethod) {
    }

    @Override
    void visitFieldAnnotation(OnFailure annotation, FieldInfo field) {
    }

    @Override
    void visitSpec(SpecInfo spec) {
    }
}

@SuppressWarnings('CatchThrowable')
class OnFailureInterceptor implements IMethodInterceptor {
    OnFailure onFailure

    OnFailureInterceptor(OnFailure onFailure) {
        this.onFailure = onFailure
    }

    @Override
    void intercept(IMethodInvocation invocation) throws Throwable {
        try {
            invocation.proceed()
        } catch (Throwable e) {
            Closure cl = onFailure.handler().newInstance(null, null)
            cl.delegate = e
            cl()
            throw e
        }
    }
}
