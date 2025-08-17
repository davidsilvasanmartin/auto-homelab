"""
Defines the Python classes for representing environment variables, and provides
some utilities for working with them
"""
from typing import List
from pathlib import Path
import sys
import secrets
import string

from app.printer import Printer
from app.validator import Validator

class EnvVar:
    name: str
    type: str
    description: str | None
    value: str | None

    def __init__(self, name: str, var_type: str, description: str | None = None, value: str | None = None) -> None:
        self.name = name
        self.type = var_type
        self.description = description
        self.value = value.strip() if isinstance(value, str) else None

    def set_value(self, value: str) -> None:
        self.value = value.strip()

    def get_dotenv_value(self) -> str:
        """Return the value to be written to .env"""
        parts: list[str] = []
        if self.description is not None:
            for line in Printer.wrap_lines(text=self.description, width=120):
                parts.append(f"# {line}")
        if self.value is not None:
                parts.append(Printer.format_dotenv_key_value(key=self.name, value=self.value))
        else:
            parts.append(f"{self.name}=")
        return "\n".join(parts)

class EnvVarsSection:
    name: str
    description: str | None
    vars: List[EnvVar]

    def __init__(self, name: str, description: str) -> None:
        self.name = name
        self.description = description
        self.vars = []

    def add_var(self, var: EnvVar) -> None:
        self.vars.append(var)

class EnvVarsRoot:
    prefix = "HOMELAB"
    sections: list[EnvVarsSection]

    def __init__(self) -> None:
        self.sections = []

    def add_section(self, section: EnvVarsSection) -> None:
        self.sections.append(section)


def _prompt(prompt_text: str) -> str | None:
    try:
        return input(prompt_text).strip()
    except KeyboardInterrupt:
        Printer.info(msg="\nAborted by user.")
        sys.exit(0)

def get_value_for_type(var_name: str, var_description: str, var_type: str, var_value:str|None) -> str:
    Printer.info(f"\n> {var_name}: {var_description or ''}")

    # Var types that DO NOT require user input
    match var_type:
        case "CONSTANT":
            Validator.validate_string_or_none(value=var_value, name=f"Variable '{var_name}' value")
            Printer.info(f"Defaulting to: {var_value}")
            return var_value
        case "GENERATED":
            # Generate a value based on the schema's "value" field, expected as "<SET>:<LENGTH>"
            charset_name = "ALPHA"
            length = 32
            charset_pools = {
                "ALL": string.ascii_letters + string.digits + "%&*+-.:<>^_|~",
                "ALPHA": string.ascii_letters + string.digits,
            }

            try:
                Validator.validate_string(value=var_value, name=f"Variable '{var_name}' generation spec")
                parts = str(var_value).split(":", 1)
                if len(parts) != 2:
                    raise ValueError("Invalid GENERATED spec format.")
                raw_set, raw_len = parts[0].strip().upper(), parts[1].strip()
                if raw_set not in charset_pools:
                    raise ValueError(f"Invalid GENERATED set. Use {charset_pools.keys()}.")
                parsed_len = int(raw_len)
                if parsed_len <= 0 or parsed_len > 1024:
                    raise ValueError("Invalid GENERATED length.")
                charset_name = raw_set
                length = parsed_len
            except Exception as e:
                Printer.info(f"Invalid GENERATED spec '{var_value}' for {var_name} ({e}). Defaulting to ALPHA:32.")

            pool = charset_pools[charset_name]
            generated = "".join(secrets.choice(pool) for _ in range(length))
            Printer.info(f"Generated a secret value of length {length} for {var_name}.")
            return generated

    # Var types that DO require user input
    while True:
        user_val = _prompt(f"Enter value for {var_name} ({var_type}): ")

        match var_type:
            case "IP":
                try:
                    Validator.validate_ip(value=user_val, name=var_name)
                    return user_val
                except ValueError:
                    Printer.info("Invalid IP address. Please enter a valid IPv4 or IPv6 address.")
                    continue
            case "STRING":
                try:
                    Validator.validate_string(value=user_val, name=var_name)
                    return user_val
                except ValueError:
                    Printer.info("Invalid string. Please enter a non-empty string.")
            case "PATH":
                # Ensure it's a non-empty string first
                Validator.validate_string(value=user_val, name=var_name)

                p = Path(user_val).expanduser()
                try:
                    if p.exists():
                        if p.is_dir():
                            return str(p.resolve())
                        else:
                            Printer.info(f"Path exists but is not a directory: {p}. Please enter a directory path.")
                            continue
                    else:
                        # Attempt to create the directory
                        p.mkdir(parents=True, exist_ok=True)
                        Printer.info(f"Created directory: {p.resolve()}")
                        return str(p.resolve())
                except Exception as e:
                    Printer.info(f"Cannot use '{p}' as a directory ({e}). Please enter a different path.")
                    continue

        raise ValueError(f"Unsupported TYPE '{var_type}' for {var_name}.")
