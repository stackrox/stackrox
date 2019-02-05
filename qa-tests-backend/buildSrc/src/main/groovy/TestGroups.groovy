class TestGroups {
	protected static final SMOKE_CATEGORY = "smoke"
	protected static final SMOKE_CATEGORY_CLASS = "groups.SMOKE"

	protected static final BAT_CATEGORY = "bat"
	protected static final BAT_CATEGORY_CLASS = "groups.BAT"

	protected static final INTEGRATION_CATEGORY = "it"
	protected static final INTEGRATION_CATEGORY_CLASS = "groups.Integration"

	protected static final ENFORCEMENT_CATEGORY = "enforcement"
	protected static final ENFORCEMENT_CATEGORY_CLASS = "groups.PolicyEnforcement"

	private static final groupDefinitions = [
 			(SMOKE_CATEGORY)          : SMOKE_CATEGORY_CLASS,
			(BAT_CATEGORY)            : BAT_CATEGORY_CLASS,
			(INTEGRATION_CATEGORY)    : INTEGRATION_CATEGORY_CLASS,
			(ENFORCEMENT_CATEGORY)    : ENFORCEMENT_CATEGORY_CLASS
	]

	private final Collection<String> groupsParam

	TestGroups(String groupsString) {
		groupsParam = (groupsString ?: "")
				.split(",")
				.toList()
				.findAll { !it.isAllWhitespace()
		}
	}

	String[] excludedGroups() {
		resolveGroups(excludes())
	}

	String[] includedGroups() {
		resolveGroups(includes())
	}

	private String[] resolveGroups(Collection<String> groups) {
		groups
				.collect { groupDefinitions[it] }
				.toArray(new String[groups.size()])
	}

	private Collection<String> includes() {
		groupsParam.findAll { !isExcluded(it) }
	}

	private Collection<String> excludes() {
		groupsParam
				.findAll { isExcluded(it) }
				.collect { it.replaceFirst("-", "") }
	}

	private boolean isExcluded(String group) {
		group.startsWith("-")
	}
}
