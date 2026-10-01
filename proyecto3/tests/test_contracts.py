"""Pruebas del contrato y del gancho; no sustituyen pruebas de servicios."""

from copy import deepcopy
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("check", ROOT / "scripts/check.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class ContractTests(unittest.TestCase):
    def setUp(self):
        self.events = json.loads((ROOT / "tests/fixtures/reports.json").read_text())
        self.expected = json.loads((ROOT / "tests/fixtures/expected.json").read_text())

    def test_known_dataset(self):
        self.assertEqual(check.aggregate(self.events), self.expected)

    def test_redelivery(self):
        self.assertEqual(check.aggregate(self.events + [self.events[2]]), self.expected)

    def test_identical_content_with_new_identity(self):
        event = deepcopy(self.events[2])
        event["event_id"] = "00000000-0000-4000-8000-000000000099"
        result = check.aggregate(self.events + [event])
        self.assertEqual(result["country_count"], 4)
        self.assertEqual(result["total"], 6)

    def test_conflicting_identity(self):
        event = deepcopy(self.events[2])
        event["report"]["warplanes_in_air"] = 21
        with self.assertRaises(ValueError):
            check.aggregate(self.events + [event])

    def test_equivalent_timestamp_on_redelivery(self):
        for equivalent in ("2026-09-29T06:00:00-06:00", "2026-09-29T12:00:00.000Z"):
            event = deepcopy(self.events[0])
            event["report"]["timestamp"] = equivalent
            self.assertEqual(check.aggregate(self.events + [event]), self.expected)

    def test_timestamp_precision(self):
        report = deepcopy(self.events[0]["report"])
        report["timestamp"] = "2026-09-29T12:00:00.123456Z"
        check.validate_report(report)
        report["timestamp"] = "2026-09-29T12:00:00.1234567Z"
        with self.assertRaises(ValueError):
            check.validate_report(report)

    def test_first_delivery_out_of_order(self):
        result = check.aggregate([self.events[4], self.events[0]])
        self.assertEqual(result["country_count"], 2)
        self.assertEqual(result["top_planes"], [["CHN", 30]])
        self.assertEqual([row[1:] for row in result["series"]], [[10, 2], [30, 8]])

    def test_empty_run(self):
        result = check.aggregate([])
        self.assertEqual(result["total"], 0)
        self.assertIsNone(result["planes"]["min"])
        self.assertEqual(result["planes"]["modes"], [])
        self.assertEqual(result["top_planes"], [])

    def test_modes_and_ranking_ties(self):
        result = check.aggregate([self.events[0], self.events[1]])
        self.assertEqual(result["planes"]["modes"], [10, 20])
        self.assertEqual(result["planes"]["frequency"], 1)
        result = check.aggregate([self.events[1], self.events[2]])
        self.assertEqual(result["top_planes"], [["CHN", 20], ["USA", 20]])

    def test_identical_timestamp_uses_identity(self):
        earlier, later = deepcopy(self.events[0]), deepcopy(self.events[4])
        later["report"]["timestamp"] = earlier["report"]["timestamp"]
        result = check.aggregate([later, earlier])
        self.assertEqual(result["country_count"], 2)
        self.assertEqual(result["top_planes"], [["CHN", 30]])

    def test_invalid_input(self):
        changes = [("country", "XXX"), ("country", "chn"),
                   ("warplanes_in_air", -1), ("warplanes_in_air", 51),
                   ("warplanes_in_air", True), ("warplanes_in_air", 10.0),
                   ("warplanes_in_air", float("nan")), ("warplanes_in_air", float("inf")),
                   ("warships_in_water", 31), ("warships_in_water", "5"),
                   ("warships_in_water", None),
                   ("timestamp", "2026-02-30T12:00:00Z"),
                   ("timestamp", "2026-09-29T12:00:00"),
                   ("timestamp", "2026-09-29T12:00:00+00:99")]
        for field, value in changes:
            with self.subTest(field=field, value=value):
                report = deepcopy(self.events[0]["report"])
                report[field] = value
                with self.assertRaises(ValueError):
                    check.validate_report(report)

    def test_missing_and_extra_fields(self):
        report = deepcopy(self.events[0]["report"])
        report["extra"] = 1
        with self.assertRaises(ValueError):
            check.validate_report(report)
        del report["extra"]
        del report["country"]
        with self.assertRaises(ValueError):
            check.validate_report(report)

    def test_bounds_and_timezone(self):
        report = deepcopy(self.events[0]["report"])
        for planes, ships in ((0, 0), (50, 30)):
            report.update(warplanes_in_air=planes, warships_in_water=ships)
            check.validate_report(report)
        self.assertEqual(check.timestamp("2026-09-29T06:00:00-06:00"),
                         check.timestamp("2026-09-29T12:00:00Z"))

    def test_hook_rejects_incorrect_result(self):
        expected = deepcopy(self.expected)
        expected["country_count"] = 999
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "incorrect.json"
            path.write_text(json.dumps(expected))
            result = subprocess.run([sys.executable, str(ROOT / "scripts/check.py"),
                                     "--expected", str(path)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertIn("estadísticas distintas", result.stderr)


if __name__ == "__main__":
    unittest.main()
