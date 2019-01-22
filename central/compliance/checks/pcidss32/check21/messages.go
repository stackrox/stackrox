package check21

const interpretationText = `StackRox does not ship with a default password. Instead, it generates a random password, or allows
a user defined password, with every deployment. Therefore, StackRox itself is in compliance.`

func passText() string {
	return "StackRox either randomly generates a strong admin password, or the user supplies one, for every deployment."
}
