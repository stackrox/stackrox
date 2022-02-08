package regocompile

/*

The following is an example of the rego we created out of a policy.

In this case, the policy is as follows:
* Volume Type: "hostpath" OR "pvc"
We also want to retrieve the container name and volume name of any volumes that match.

Note the following things:
* We need to tell rego how to go from a label like Volume Type to the actual path within the input.
  Where we need to traverse arrays, we need to declare an index (using the "some" keyword), and index
  into the array with the "some" keyword. What rego then does is try all valid values of the index and
  return everything for which the query returns true.
* We create a rule called "violations" which helps define the violations. Note that in rego, there is no OR keyword;
  to get the OR semantics, you just repeat the rule, and in each rule body, you define one of the matching conditions.
  Similarly, rego has no AND keyword; to get AND semantics in a rule, you just put the statements one after the other in
  a rule. A rule only matches if every statement in it evalutes to true.
* To decompose the program, we create functions for each individual match object. The function returns an object of the
  form { "match": <bool>, "values": <array> }. Then, in the rule, we assert that "match" is true, and we return the array
  of matching values outside of rego.
* Thus, the final returned value will be a []map[string][]interface{}, with one element for each set of variable bindings
  that satisfies the query. Each set of variable bindings will have a value of each field (by key label).

package policy.main

matchVolume_TypeTo0r_hostpath(val) = result {
	result := { "match": regex.match(`^(?i:hostpath)$`, val), "values": [val] }
}

matchVolume_TypeTo0r_pvc(val) = result {
	result := { "match": regex.match(`^(?i:pvc)$`, val), "values": [val] }
}

matchAllContainer_Name(val) = result {
	result := { "match": true, "values": [val] }}

matchAllVolume_Name(val) = result {
	result := { "match": true, "values": [val] }
}

violations[result] {
	some idx0
	some idx1
	matchVolume_TypeTo0r_hostpathResult := matchVolume_TypeTo0r_hostpath(input.Containers[idx0].Volumes[idx1].Type)
	matchVolume_TypeTo0r_hostpathResult["match"]
	matchAllContainer_NameResult := matchAllContainer_Name(input.Containers[idx0].Name)
	matchAllContainer_NameResult["match"]
	matchAllVolume_NameResult := matchAllVolume_Name(input.Containers[idx0].Volumes[idx1].Name)
	matchAllVolume_NameResult["match"]
	result := {
			"Volume Type": matchVolume_TypeTo0r_hostpathResult["values"],
			"Container Name": matchAllContainer_NameResult["values"],
			"Volume Name": matchAllVolume_NameResult["values"]
	}
}

violations[result] {
	some idx0
	some idx1
	matchVolume_TypeTo0r_pvcResult  := matchVolume_TypeTo0r_pvc(input.Containers[idx0].Volumes[idx1].Type)
	matchVolume_TypeTo0r_pvcResult["match"]
	matchAllContainer_NameResult := matchAllContainer_Name(input.Containers[idx0].Name)
	matchAllContainer_NameResult["match"]
	matchAllVolume_NameResult := matchAllVolume_Name(input.Containers[idx0].Volumes[idx1].Name)
	matchAllVolume_NameResult["match"]
	result := {
			"Volume Type": matchVolume_TypeTo0r_pvcResult["values"],
			"Container Name": matchAllContainer_NameResult["values"],
			"Volume Name": matchAllVolume_NameResult["values"]
	}
}
*/
