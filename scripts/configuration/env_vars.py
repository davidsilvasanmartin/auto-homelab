"""
Defines the Python classes for representing environment variables, and provides
some utilities for working with them
"""

import secrets
import string
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Protocol

from scripts.printer import Printer
from scripts.validator import Validator

# -----------------------------
# Domain model (Composite root)
# -----------------------------


@dataclass
class EnvVar:
    name: str
    type: str
    description: str
    value: str | None = None

    def set_value(self, value: str) -> None:
        self.value = value.strip()


@dataclass
class EnvVarsSection:
    name: str
    description: str | None = None
    vars: list[EnvVar] = field(default_factory=list[EnvVar])

    def add_var(self, var: EnvVar) -> None:
        self.vars.append(var)


@dataclass
class EnvVarsRoot:
    prefix: str
    sections: list[EnvVarsSection] = field(default_factory=list[EnvVarsSection])

    def add_section(self, section: EnvVarsSection) -> None:
        self.sections.append(section)


# -----------------------------
# Prompt and UI helpers
# -----------------------------


def _prompt(prompt_text: str) -> str:
    try:
        return input(prompt_text).strip()
    except KeyboardInterrupt:
        Printer.info(msg="\nAborted by user.")
        sys.exit(0)


# -----------------------------
# Strategy: acquire variable value by type
# -----------------------------


class VarTypeStrategy(Protocol):
    """
    Strategy interface to acquire a value for an environment variable type.
    """

    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str: ...


class ConstantStrategy:
    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str:
        # TODO we are always printing the following line, consider abstracting this away from particular implementations
        Printer.info(f"\n> {var.name}: {var.description}")
        Validator.validate_string_or_none(value=default_spec, name=f"Variable '{var.name}' value")
        Printer.info(f"Defaulting to: {default_spec}")
        return default_spec or ""


class GeneratedStrategy:
    """
    default_spec format: "<SET>:<LENGTH>"
    SET in {"ALL", "ALPHA"}
    """

    _charset_pools: dict[str, str] = {
        "ALL": string.ascii_letters + string.digits + "%&*+-.:<>^_|~",
        "ALPHA": string.ascii_letters + string.digits,
    }

    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str:
        Printer.info(f"\n> {var.name}: {var.description}")
        charset_name = "ALPHA"
        length = 32

        try:
            Validator.validate_string(value=default_spec, name=f"Variable '{var.name}' generation spec")
            parts = str(default_spec).split(":", 1)
            if len(parts) != 2:
                raise ValueError("Invalid GENERATED spec format.")
            raw_set, raw_len = parts[0].strip().upper(), parts[1].strip()
            if raw_set not in self._charset_pools:
                raise ValueError(f"Invalid GENERATED set. Use {set(self._charset_pools.keys())}.")
            parsed_len = int(raw_len)
            if parsed_len <= 0 or parsed_len > 1024:
                raise ValueError("Invalid GENERATED length.")
            charset_name = raw_set
            length = parsed_len
        except Exception as e:
            Printer.info(
                f"Invalid GENERATED spec '{default_spec}' for {var.name} ({e}). Defaulting to ALPHA:32."
            )

        pool = self._charset_pools[charset_name]
        generated = "".join(secrets.choice(pool) for _ in range(length))
        Printer.info(f"Generated a secret value of length {length} for {var.name}.")
        return generated


class IpStrategy:
    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str:
        Printer.info(f"\n> {var.name}: {var.description}")
        while True:
            user_val = _prompt(f"Enter value for {var.name} (IP): ")
            try:
                Validator.validate_ip(value=user_val, name=var.name)
                return user_val
            except ValueError:
                Printer.info("Invalid IP address. Please enter a valid IPv4 or IPv6 address.")


class StringStrategy:
    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str:
        Printer.info(f"\n> {var.name}: {var.description}")
        while True:
            user_val = _prompt(f"Enter value for {var.name} (STRING): ")
            try:
                Validator.validate_string(value=user_val, name=var.name)
                return user_val
            except ValueError:
                Printer.info("Invalid string. Please enter a non-empty string.")


class PathStrategy:
    def acquire(self, *, var: EnvVar, default_spec: str | None) -> str:
        Printer.info(f"\n> {var.name}: {var.description}")
        while True:
            user_val = _prompt(f"Enter value for {var.name} (PATH): ")
            # Ensure it's a non-empty string first
            Validator.validate_string(value=user_val, name=var.name)

            p = Path(user_val).expanduser()
            try:
                if p.exists():
                    if p.is_dir():
                        return str(p.resolve())
                    else:
                        Printer.info(
                            f"Path exists but is not a directory: {p}. Please enter a directory path."
                        )
                        continue
                else:
                    # Attempt to create the directory
                    p.mkdir(parents=True, exist_ok=True)
                    Printer.info(f"Created directory: {p.resolve()}")
                    return str(p.resolve())
            except Exception as e:
                Printer.info(f"Cannot use '{p}' as a directory ({e}). Please enter a different path.")


# -----------------------------
# Factory Method: registry for strategies
# -----------------------------


class TypeHandlerRegistry:
    """
    Central registry for variable type strategies.
    - Register new types with register("TYPE_NAME", strategy_instance)
    - Lookup via get("TYPE_NAME")
    """

    def __init__(self) -> None:
        self._registry: dict[str, VarTypeStrategy] = {}

    def register(self, type_name: str, strategy: VarTypeStrategy) -> None:
        self._registry[type_name.upper()] = strategy

    def get(self, type_name: str) -> VarTypeStrategy:
        type_key = type_name.upper()
        if type_key not in self._registry:
            raise ValueError(f"Unsupported TYPE '{type_name}'.")
        return self._registry[type_key]


# Default registry instance preloaded with built-ins
_default_registry = TypeHandlerRegistry()
_default_registry.register("CONSTANT", ConstantStrategy())
_default_registry.register("GENERATED", GeneratedStrategy())
_default_registry.register("IP", IpStrategy())
_default_registry.register("STRING", StringStrategy())
_default_registry.register("PATH", PathStrategy())


def get_value_for_type(
    var_name: str,
    var_description: str,
    var_type: str,
    var_value: str | None,
) -> str:
    """
    Acquire a value for a variable by delegating to the appropriate type strategy.
    """
    var = EnvVar(name=var_name, type=var_type, description=var_description, value=var_value)
    strategy = _default_registry.get(var_type)
    return strategy.acquire(var=var, default_spec=var_value)
