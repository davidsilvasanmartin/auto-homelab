import pytest

from scripts.printer import Printer


class TestPrinterWrapLines:
    def test_wrap_lines_none(self):
        assert Printer.wrap_lines(None) == [""]

    def test_wrap_lines_empty_string(self):
        assert Printer.wrap_lines("") == [""]

    @pytest.mark.parametrize(
        "text,width,expected",
        [
            ("short line", 120, ["short line"]),
            ("a" * 10, 10, ["a" * 10]),
            ("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 5, ["ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY", "Z"]),
            ("x" * 130, 50, ["x" * 50, "x" * 50, "x" * 30]),
        ],
    )
    def test_wrap_lines_exact_width(self, text, width, expected):
        assert Printer.wrap_lines(text, width=width) == expected

    def test_wrap_lines_long_sentence_wraps(self):
        text = (
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
            "Vestibulum vulputate, sapien non hendrerit commodo, "
            "nisi orci dictum justo, non iaculis turpis lacus a est."
        )
        width = 60
        lines = Printer.wrap_lines(text, width=width)
        assert lines == [
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ",
            "Vestibulum vulputate, sapien non hendrerit commodo, nisi ",
            "orci dictum justo, non iaculis turpis lacus a est.",
        ]

    def test_wrap_lines_multiple_paragraphs_and_blank_lines(self):
        text = "para one line 1\n\npara two line 1\npara two line 2"
        lines = Printer.wrap_lines(text, width=80)
        assert lines == [
            "para one line 1",
            "",
            "para two line 1",
            "para two line 2",
        ]

    def test_wrap_lines_whitespace_only_paragraph(self):
        text = "First\n   \nLast"
        lines = Printer.wrap_lines(text, width=80)
        assert lines == [
            "First",
            "   ",
            "Last",
        ]


class TestPrinterInfo:
    def test_info_prints_wrapped_lines(self, capsys: pytest.CaptureFixture[str]):
        text = "This is a test string that will be wrapped appropriately to a given width."
        Printer.info(text)
        captured = capsys.readouterr()
        expected_lines = Printer.wrap_lines(text=text, width=120)
        expected = "\n".join(expected_lines) + "\n"
        assert captured.out == expected

    def test_info_handles_none_as_blank_line(self, capsys: pytest.CaptureFixture[str]):
        Printer.info(None)  # type: ignore[arg-type]
        captured = capsys.readouterr()
        assert captured.out == "\n"

    def test_info_handles_empty_string(self, capsys: pytest.CaptureFixture[str]):
        Printer.info("")
        captured = capsys.readouterr()
        assert captured.out == "\n"


class TestFormatDotenvKeyValue:
    def test_basic_key_value(self):
        assert Printer.format_dotenv_key_value("KEY", "VALUE") == 'KEY="VALUE"'

    def test_value_with_double_quotes_is_escaped(self):
        assert (
            Printer.format_dotenv_key_value("GREETING", 'He said "hello"') == 'GREETING="He said \\"hello\\""'
        )

    def test_empty_string_value(self):
        assert Printer.format_dotenv_key_value("EMPTY", "") == 'EMPTY=""'

    def test_none_value_becomes_empty_string(self):
        assert (
            Printer.format_dotenv_key_value("NONE_TO_EMPTY", None)  # type: ignore[arg-type]
            == 'NONE_TO_EMPTY=""'
        )

    def test_key_is_stripped(self):
        assert Printer.format_dotenv_key_value("  TRIMMED_KEY  ", "v") == 'TRIMMED_KEY="v"'

    def test_invalid_key_empty_string_raises(self):
        with pytest.raises(ValueError):
            Printer.format_dotenv_key_value("", "VALUE")

    def test_invalid_key_whitespace_only_raises(self):
        with pytest.raises(ValueError):
            Printer.format_dotenv_key_value("   ", "VALUE")

    def test_value_with_backslashes_and_quotes(self):
        assert (
            Printer.format_dotenv_key_value("PATH", 'C:\\Program Files\\App\\"bin"')
            == 'PATH="C:\\Program Files\\App\\\\"bin\\""'
        )

    def test_value_with_newline_preserved(self):
        assert Printer.format_dotenv_key_value("MULTI", "line1\nline2") == 'MULTI="line1\nline2"'

    def test_non_string_value_is_stringified(self):
        assert (
            Printer.format_dotenv_key_value("NUMBER", 123)  # type: ignore[arg-type]
            == 'NUMBER="123"'
        )
