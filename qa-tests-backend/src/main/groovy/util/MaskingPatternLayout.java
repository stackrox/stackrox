// copied from https://howtodoinjava.com/logback/masking-sensitive-data/
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
		//When masking is disabled in a environment
		if (appliedPattern == null) {
			return message;
		}
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
