import ch.qos.logback.classic.encoder.PatternLayoutEncoder

appender("CONSOLE", ConsoleAppender) {
    encoder(PatternLayoutEncoder) {
        pattern = "%-4relative [%thread] - %msg%n"
    }
}

// Limit logs output by libraries we use (like spock-reports) to just errors.
// See https://stackoverflow.com/questions/13867057/how-to-use-logback-configured-via-logback-groovy-with-groovy
// for details on how to configure this.
root(ERROR, ["CONSOLE"])
