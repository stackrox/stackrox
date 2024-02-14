package sampleScripts

import common.Constants
import util.Helpers

// Unit tests for Helpers.compareAnnotations

Map<String, String> a = new HashMap<String, String>() {{
    put("same", "value")
}}

Map<String, String> b = new HashMap<String, String>() {{
    put("same", "value")
}}

assert Helpers.compareAnnotations(a, b)

a.put("a_only_key", "value")
assert !Helpers.compareAnnotations(a, b)

b.put("b_only_key", "value")
assert !Helpers.compareAnnotations(a, b)

a.remove("a_only_key")
assert !Helpers.compareAnnotations(a, b)

b.remove("b_only_key")
assert Helpers.compareAnnotations(a, b)

a.put("different", "a")
b.put("different", "b")
assert !Helpers.compareAnnotations(a, b)
a.remove("different")
b.remove("different")

Random random = new Random();

String generated512String = random.ints(48, 122 + 1)
      .limit(512)
      .collect(StringBuilder::new, StringBuilder::appendCodePoint, StringBuilder::append)
      .toString();

a.put("truncated", generated512String)
b.put("truncated", generated512String)
assert Helpers.compareAnnotations(a, b)

// long orchestrator annotations are OK when different in stackrox
b.put("truncated", generated512String.substring(0, 257) + "...")
assert Helpers.compareAnnotations(a, b)

// length is up to a limit
a.put("truncated", generated512String.substring(0, Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH - 1))
assert !Helpers.compareAnnotations(a, b)
