import importlib
import pytest

patch_csv = importlib.import_module("patch-csv")

XyzVersion = patch_csv.XyzVersion


@pytest.mark.parametrize("current,first,previous,skips,expected", [
    ("1.0.0", "4.0.0", "0.0.0", [], None),
    ("4.0.0", "4.0.0", "3.74.0", [], None),
    ("4.1.3", "4.0.0", "4.1.0", [], "4.1.2"),
    ("4.1.1", "4.0.0", "4.1.0", [], "4.1.0"),
    ("4.1.1", "4.0.0", "4.1.0", ["4.1.0"], "4.1.0"),
    ("4.2.0", "4.0.0", "4.1.0", ["4.1.0"], "4.1.1"),
    ("4.3.0", "4.0.0", "4.2.0", ["4.1.0", "4.2.1", "4.4.0"], "4.2.0"),
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
