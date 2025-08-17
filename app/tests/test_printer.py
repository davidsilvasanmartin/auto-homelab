# tests/test_printer.py
import io
import unittest
from contextlib import redirect_stdout

from app.printer import Printer


class TestPrinterWrapLines(unittest.TestCase):
    def test_wrap_lines_none(self):
        self.assertEqual(Printer.wrap_lines(None), [""])

    def test_wrap_lines_empty_string(self):
        self.assertEqual(Printer.wrap_lines(""), [""])

    def test_wrap_lines_single_short_line_no_wrap(self):
        text = "short line"
        lines = Printer.wrap_lines(text, width=120)
        self.assertEqual(lines, [text])

    def test_wrap_lines_exact_width(self):
        text = "abcdefghij"  # length 10
        lines = Printer.wrap_lines(text, width=10)
        self.assertEqual(lines, [text])

    def test_wrap_lines_long_single_word_wraps(self):
        # Long word without spaces should be split
        text = "x" * 130
        width = 50
        lines = Printer.wrap_lines(text, width=width)
        self.assertEqual(lines, [
            "x" * 50,
            "x" * 50,
            "x" * 30,
        ])

    def test_wrap_lines_long_sentence_wraps(self):
        text = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " \
               "Vestibulum vulputate, sapien non hendrerit commodo, " \
               "nisi orci dictum justo, non iaculis turpis lacus a est."
        width = 60
        lines = Printer.wrap_lines(text, width=width)
        self.assertEqual(lines, [
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ",
            "Vestibulum vulputate, sapien non hendrerit commodo, nisi ",
            "orci dictum justo, non iaculis turpis lacus a est."
        ])

    def test_wrap_lines_multiple_paragraphs_and_blank_lines(self):
        text = "para one line 1\n\npara two line 1\npara two line 2"
        lines = Printer.wrap_lines(text, width=80)
        self.assertEqual(lines, [
            "para one line 1",
            "",
            "para two line 1",
            "para two line 2",
        ])

    def test_wrap_lines_whitespace_only_paragraph(self):
        text = "First\n   \nLast"
        lines = Printer.wrap_lines(text, width=80)
        self.assertEqual(lines, [
            "First",
            "   ",
            "Last"
        ])

    def test_wrap_lines_custom_width_small(self):
        text = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        width = 5
        lines = Printer.wrap_lines(text, width=width)
        self.assertEqual(lines, [
            "ABCDE",
            "FGHIJ",
            "KLMNO",
            "PQRST",
            "UVWXY",
            "Z"
        ])


class TestPrinterInfo(unittest.TestCase):
    def test_info_prints_wrapped_lines(self):
        text = "This is a test string that will be wrapped appropriately to a given width."
        f = io.StringIO()
        with redirect_stdout(f):
            Printer.info(text)
        printed = f.getvalue()

        expected_lines = Printer.wrap_lines(text=text, width=120)
        expected = "\n".join(expected_lines) + "\n"  # print adds newline per line
        self.assertEqual(printed, expected)

    def test_info_handles_none_as_blank_line(self):
        f = io.StringIO()
        with redirect_stdout(f):
            Printer.info(None)  # type: ignore[arg-type]
        printed = f.getvalue()
        # One blank line
        self.assertEqual(printed, "\n")

    def test_info_handles_empty_string(self):
        f = io.StringIO()
        with redirect_stdout(f):
            Printer.info("")
        printed = f.getvalue()
        # One blank line
        self.assertEqual(printed, "\n")


class TestFormatDotenvKeyValue(unittest.TestCase):
    def test_basic_key_value(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("KEY", "VALUE"),
            'KEY="VALUE"',
        )

    def test_value_with_double_quotes_is_escaped(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("GREETING", 'He said "hello"'),
            'GREETING="He said \\"hello\\""',
        )

    def test_empty_string_value(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("EMPTY", ""),
            'EMPTY=""',
        )

    def test_none_value_becomes_empty_string(self):
        # Even though type hint says str, function tolerates None and treats it as empty
        self.assertEqual(
            Printer.format_dotenv_key_value("NONE_TO_EMPTY", None),  # type: ignore[arg-type]
            'NONE_TO_EMPTY=""',
        )

    def test_key_is_stripped(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("  TRIMMED_KEY  ", "v"),
            'TRIMMED_KEY="v"',
        )

    def test_invalid_key_empty_string_raises(self):
        with self.assertRaises(ValueError):
            Printer.format_dotenv_key_value("", "VALUE")

    def test_invalid_key_whitespace_only_raises(self):
        with self.assertRaises(ValueError):
            Printer.format_dotenv_key_value("   ", "VALUE")

    def test_value_with_backslashes_and_quotes(self):
        # Ensure only quotes are escaped; backslashes remain intact
        self.assertEqual(
            Printer.format_dotenv_key_value("PATH", 'C:\\Program Files\\App\\"bin"'),
            'PATH="C:\\Program Files\\App\\\\"bin\\""',
        )

    def test_value_with_newline_preserved(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("MULTI", "line1\nline2"),
            'MULTI="line1\nline2"',
        )

    def test_non_string_value_is_stringified(self):
        self.assertEqual(
            Printer.format_dotenv_key_value("NUMBER", 123),  # type: ignore[arg-type]
            'NUMBER="123"',
        )


if __name__ == "__main__":
    unittest.main()