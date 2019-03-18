package util

class Timer {
    private Integer currIteration = 0
    private final Integer iterations
    private final Integer delayMilliseconds
    private final long startTime

    Timer(Integer iterations, Integer delaySeconds) {
        this.iterations = iterations
        this.delayMilliseconds = delaySeconds * 1000
        this.startTime = System.currentTimeMillis()
    }

    Boolean IsValid() {
        if (currIteration == 0) {
            currIteration++
            return true
        }
        if (currIteration == this.iterations) {
            return false
        }
        sleep(this.delayMilliseconds)
        currIteration++
        return true
    }

    int SecondsSince() {
        return (System.currentTimeMillis() - this.startTime) / 1000
    }
}
