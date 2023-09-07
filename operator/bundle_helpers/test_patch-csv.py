import importlib
import pytest

patch_csv = importlib.import_module("patch-csv")

XyzVersion = patch_csv.XyzVersion


@pytest.mark.parametrize("current,first,previous,skips,expected", [
    # Downstream trunk builds get no replace since their version is too small.
    ("1.0.0", "4.0.0", "0.0.0", [], None),
    # First 4.0.0 release gets no replace despite previous Y-Stream.
    ("4.0.0", "4.0.0", "3.74.0", [], None),

    # Patch follows normal release in absence of replaces.
    ("4.0.1", "4.0.0", "3.74.0", [], "4.0.0"),
    # Normal Y-Stream release replaces previous Y-Stream.
    ("4.2.0", "4.0.0", "4.1.0", [], "4.1.0"),
    # Normal patch, no skips, replaces previous patch.
    ("4.1.3", "4.0.0", "4.0.0", [], "4.1.2"),
    # Normal first patch, no skips, replaces its Y-Stream.
    ("4.1.1", "4.0.0", "4.0.0", [], "4.1.0"),

    # When the skipped patch is immediately preceding, it is still taken for replacement.
    ("4.1.1", "4.0.0", "4.0.0", ["4.1.0"], "4.1.0"),
    ("4.1.3", "4.0.0", "4.0.0", ["4.1.2"], "4.1.2"),
    # When the previous Y-Stream is skipped, we'd target the next patch.
    ("4.2.0", "4.0.0", "4.1.0", ["4.1.0"], "4.1.1"),
    # When skips don't hit, they play no role.
    ("4.3.0", "4.0.0", "4.2.0", ["4.1.0", "4.2.1", "4.4.0"], "4.2.0"),
    # It should keep iterating patches until it finds the one that's not skipped.
    ("4.3.0", "4.0.0", "4.2.0", ["4.1.0", "4.2.0", "4.2.1", "4.2.2", "4.4.0"], "4.2.3"),
])
def test_calculate_replaced_version(current, first, previous, skips, expected):
    skips = {XyzVersion.parse_from(s) for s in skips}

    result = patch_csv.calculate_replaced_version(current, first, previous, skips)

    expected = XyzVersion.parse_from(expected) if expected is not None else None
    assert result == expected


@pytest.mark.parametrize("spec,raw_name,expected", [
    (
            {
                "skips": ["rhacs-operator.v4.1.0", "rhacs-operator.v3.72.0", "rhacs-operator.v3.62.2"]
            },
            "rhacs-operator",
            {XyzVersion(4, 1, 0), XyzVersion(3, 62, 2), XyzVersion(3, 72, 0)}
    ),
    ({"skips": []}, "blah", set()),
    ({}, "blah", set()),
])
def test_parse_skips(spec, raw_name, expected):
    result = patch_csv.parse_skips(spec, raw_name)
    assert result == expected


def test_parse_skips_exception():
    with pytest.raises(RuntimeError, match="odd_one.* does not begin with different_one"):
        patch_csv.parse_skips({"skips": ["odd_one.v1.2.3"]}, "different_one")
