package util

import groovy.transform.CompileStatic

@CompileStatic
class Timer {
    private Integer currIteration = 0
    private final Integer iterations
    private final Integer delayMilliseconds
    private final long startTime

    Timer(Integer retries, Integer delaySeconds) {
        // iterations = retries + 1 because the first attempt happens immediately
        this.iterations = retries + 1
        this.delayMilliseconds = delaySeconds * 1000
        this.startTime = System.currentTimeMillis()
    }

    Boolean IsValid() {
        if (currIteration == 0) {
            currIteration++
            return true
        }
        if (currIteration >= this.iterations) {
            return false
        }
        sleep(this.delayMilliseconds)
        currIteration++
        return true
    }

    int SecondsSince() {
        return ((System.currentTimeMillis() - this.startTime) / 1000) as int
    }
}
