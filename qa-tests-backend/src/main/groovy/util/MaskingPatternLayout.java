// modified from https://howtodoinjava.com/logback/masking-sensitive-data/
package util;

import java.util.ArrayList;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import ch.qos.logback.classic.PatternLayout;
import ch.qos.logback.classic.spi.ILoggingEvent;

public class MaskingPatternLayout extends PatternLayout
{
    private Pattern appliedPattern;
    private List<String> maskPatterns = new ArrayList<>();

    public MaskingPatternLayout() {
        addMaskPattern("BEGIN (PRIVATE KEY.*)END PRIVATE KEY");
        addMaskPattern("----BEGIN ([A-Z ]*PRIVATE KEY.*)END[A-Z ]* PRIVATE KEY----");
        addMaskPattern("(LUJFR0lOIFBSSVZBVEUgS.*)");
        addMaskPattern("(LS1CRUdJTiBQUklWQVRFI.*)");
        addMaskPattern("(LS0tQkVHSU4gUFJJVkFUR.*)");
        addMaskPattern("(?i)access_key_id[:= ]*(.*)");
        addMaskPattern("(?i)secret_access_key[:= ]*(.*)");
        addMaskPattern("(?i)A[KS]IA([2-7A-Z]{16})");
        addMaskPattern("([A-Za-z0-9/+]{40})");
        addMaskPattern("((F(QoDYXd|woGZXIvYXd)|[AI]goJb3JpZ2luX2V)[A-Za-z0-9+/]+={0,2})");
    }

    public void addMaskPattern(String maskPattern) {
        maskPatterns.add(maskPattern);
        appliedPattern = Pattern.compile( maskPatterns.stream()
                .collect(Collectors.joining("|")), Pattern.MULTILINE);
    }
    @Override
    public String doLayout(ILoggingEvent event) {
        return maskMessage(super.doLayout(event));
    }
    private String maskMessage(String message) {
        StringBuilder sb = new StringBuilder(message);
        Matcher matcher = appliedPattern.matcher(sb);
        while (matcher.find()) {
            IntStream.rangeClosed(1, matcher.groupCount()).forEach(group -> {
                if (matcher.group(group) != null) {
                    IntStream.range(matcher.start(group),
                            matcher.end(group)).forEach(i -> sb.setCharAt(i, 'R'));
                }
            });
        }
        return sb.toString();
    }
}
