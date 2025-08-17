import textwrap
from app.validator import Validator

class Printer:
    """
    Utility class for printing formatted text
    """

    @staticmethod
    def wrap_lines(text: str | None, width: int = 120) -> list[str]:
        """
        Wraps text to the specified width
        :param text: The text to wrap
        :param width: The width to wrap to
        :return: The wrapped text as a list of lines
        """
        if text is None or str(text).strip() == "":
            return [""]
        lines: list[str] = []
        for paragraph in str(text).splitlines():
            if paragraph == "":
                lines.append("")
                continue
            wrapped = textwrap.wrap(
                paragraph,
                width=width,
                replace_whitespace=False,
                drop_whitespace=False,
            )
            # If wrap returns empty (e.g., very long whitespace), still print a blank line
            if not wrapped:
                lines.append("")
            else:
                lines.extend(wrapped)
        return lines

    @staticmethod
    def info(msg: str) -> None:
        """
        Prints formatted info message
        :param msg: The message to print
        """
        for line in Printer.wrap_lines(text=msg):
            print(line)

    @staticmethod
    def format_dotenv_key_value(key: str, value: str | None) -> str:
        """
        Format a key-value pair for a .env file as: KEY="VALUE"
        Double quotes inside VALUE are escaped as \"
        """
        Validator.validate_string(key, "key")
        value_str = "" if value is None else str(value)
        value_str = value_str.replace('"', r'\"')
        return f'{key.strip()}="{value_str}"'
