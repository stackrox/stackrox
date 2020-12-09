package util;

public class E2ETestException extends RuntimeException {
    public E2ETestException(String message) {
        super(message);
    }

    public E2ETestException(String message, Throwable cause) {
        super(message, cause);
    }

    public E2ETestException(Throwable cause) {
        super(cause);
    }

    public E2ETestException(String message, Throwable cause, boolean enableSuppression, boolean writableStackTrace) {
        super(message, cause, enableSuppression, writableStackTrace);
    }
}
