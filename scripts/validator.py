import ipaddress
from typing import Any


class Validator:
    """
    Utility class for validating values
    """

    @staticmethod
    def validate_string(value: Any, name: str):
        """
        Validates that the value is a non-empty string.
        :param value: The value to be validated
        :param name: The name of the value
        """
        if not isinstance(value, str) or str(value).strip() == "":
            raise ValueError(f"{name} must be a non-empty string.")

    @staticmethod
    def validate_string_or_none(value: Any, name: str):
        """
        Validates that the value is a string or null
        :param value: The value to be validated
        :param name: The name of the value
        """
        if value is not None and not isinstance(value, str):
            raise ValueError(f"{name} must be a string or null.")

    @staticmethod
    def validate_ip(value: Any, name: str):
        """
        Validates that the value is a valid IPv4 or IPv6 address
        :param value: The value to be validated
        :param name: The name of the value
        """
        try:
            ipaddress.ip_address(value)
        except ValueError as e:
            raise ValueError(f"{name} must be a valid IPv4 or IPv6 address.") from e

    @staticmethod
    def validate_dict(value: Any, name: str):
        """
        Validates that the value is a dictionary
        :param value: The value to be validated
        :param name: The name of the value
        """
        if not isinstance(value, dict):
            raise ValueError(f"{name} must be a dictionary.")

    @staticmethod
    def validate_value_in_set(value: Any, name: str, valid_values: set[Any]):
        """
        Validates that the value is in the specified set of valid values
        :param value: The value to be validated
        :param name: The name of the value
        :param valid_values: The set of valid values
        """
        if value not in valid_values:
            raise ValueError(f"{name} must be one of {valid_values}.")
