import org.spockframework.runtime.model.parallel.ExecutionMode

runner {
  parallel {
    enabled true
    // Specifications run concurrently by default
    defaultSpecificationExecutionMode ExecutionMode.CONCURRENT
    // Features run serially by default
    defaultExecutionMode ExecutionMode.SAME_THREAD
    fixed(8)
  }
}
