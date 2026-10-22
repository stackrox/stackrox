"""Exercise path facts and Git collection independently of job policy."""

import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from path_facts import classify, changed_files


class PathFactsTest(unittest.TestCase):
    def test_path_quantifiers(self):
        cases = {
            "empty diff": ([], {"any": [], "every": []}),
            "UI only": (["ui/App.tsx"], {"any": ["ui"], "every": ["ui"]}),
            "mixed areas": (["ui/App.tsx", "central/a.go"], {"any": ["ui", "go"], "every": []}),
            "newline suffix": (["a.go\n"], {"any": [], "every": []}),
            "overlapping patterns": (["ui/tool.go"], {"any": ["ui", "go"], "every": ["ui", "go"]}),
        }
        for name, (files, expected) in cases.items():
            with self.subTest(name=name):
                self.assertEqual(classify(files, {"ui": ["^ui/"], "go": [r"\.go$"]}), expected)

    def test_invalid_configuration_fails(self):
        for patterns in ({"bad": ["["]}, {"bad": []}, {"bad": [None]}):
            with self.subTest(patterns=patterns), self.assertRaises((ValueError, TypeError, re.error)):
                classify(["ui/a"], patterns)

    def test_missing_commits_fail_instead_of_becoming_an_empty_diff(self):
        with self.assertRaises(ValueError):
            changed_files("", "")

    def test_path_collection_has_no_workflow_filter_file_limit(self):
        files = [f"docs/{index}.md" for index in range(4000)] + ["central/a.go"]
        self.assertEqual(classify(files, {"go": [r"\.go$"]}), {"any": ["go"], "every": []})

    def test_git_diff_handles_deletions_renames_and_unusual_paths(self):
        with tempfile.TemporaryDirectory() as temp:
            repo = Path(temp)

            def git(*args):
                return subprocess.check_output(["git", "-C", temp, *args], stderr=subprocess.DEVNULL).decode().strip()

            git("init", "-b", "main")
            git("config", "user.email", "ci@example.invalid")
            git("config", "user.name", "CI fixture")
            git("config", "commit.gpgSign", "false")
            (repo / "old.go").write_text("package example\n")
            (repo / "gone.md").write_text("delete me\n")
            git("add", "old.go", "gone.md")
            git("commit", "-m", "Initial fixture")
            base = git("rev-parse", "HEAD")
            (repo / "old.go").rename(repo / "new.go")
            (repo / "gone.md").unlink()
            (repo / "line\nbreak.go").write_text("package example\n")
            git("add", "old.go", "gone.md", "new.go", "line\nbreak.go")
            git("commit", "-m", "Change fixture paths")
            head = git("rev-parse", "HEAD")
            script = Path(__file__).with_name("path_facts.py").resolve()
            env = os.environ | {"CI_BASE": base, "CI_HEAD": head}
            result = subprocess.check_output(["python3", str(script)], cwd=temp, env=env).decode()
            self.assertIn("known=true", result)
            self.assertIn("go", json.loads(result.splitlines()[0].split("=", 1)[1]))
            self.assertEqual(set(git("diff", "--name-only", "-z", f"{base}...{head}").split("\0")) - {""},
                             {"new.go", "gone.md", "line\nbreak.go"})
