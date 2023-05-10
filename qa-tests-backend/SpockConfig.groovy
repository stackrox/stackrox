import org.spockframework.runtime.model.parallel.ExecutionMode

runner {
  parallel {
    enabled true
    fixed(2)
    defaultSpecificationExecutionMode ExecutionMode.CONCURRENT
    defaultExecutionMode ExecutionMode.SAME_THREAD
  }
}
