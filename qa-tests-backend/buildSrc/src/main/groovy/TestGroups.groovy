class TestGroups {
    private String[] excludedGroups
    private String[] includedGroups

	TestGroups(String groupsString) {
		Collection<String> groups = (groupsString ?: "")
				.split(",")
				.toList()
				.findAll { !it.isAllWhitespace() }
        includedGroups = resolveGroups(groups.findAll { !isExcluded(it) })
        // Check for the - at the beginning, and remove it before passing the string.
        excludedGroups = resolveGroups(groups.findAll { isExcluded(it) }.collect { it.substring(1) })
	}

	String[] getExcludedGroups() {
        return excludedGroups
	}

	String[] getIncludedGroups() {
        return includedGroups
	}

	private static String[] resolveGroups(Collection<String> groups) {
        // There's some Groovy magic at play here. All our groups are tagged by (empty) classes defined
        // in groups.groovy. Groovy uses reflection to match those class names against the strings passed here.
        // We add a "groups." at the beginning so that the user just needs to pass, say, "BAT", and
        // we translate it to groups.BAT
		groups.collect { "groups." + it }.toArray(new String[groups.size()])
	}

	private static boolean isExcluded(String group) {
		group.startsWith("-")
	}
}
