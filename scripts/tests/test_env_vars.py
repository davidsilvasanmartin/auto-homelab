from scripts.configuration.env_vars import EnvVar


class TestEnvVarSetValue:
    def test_set_value_strips_whitespace(self):
        var = EnvVar(name="MY_KEY", var_type="STRING", value=None)
        var.set_value("  value with spaces  ")
        assert var.value == "value with spaces"

    def test_constructor_strips_initial_value(self):
        var = EnvVar(name="INIT_KEY", var_type="STRING", value="  initial  ")
        assert var.value == "initial"

    def test_none_description_and_value(self):
        var = EnvVar(name="NO_DESC_NO_VAL", var_type="STRING")
        var.set_value(None)
        assert var.get_dotenv_value() == "NO_DESC_NO_VAL="


class TestEnvVarGetDotenvValue:
    def test_no_description_no_value(self):
        var = EnvVar(name="NO_DESC_NO_VAL", var_type="STRING")
        assert var.get_dotenv_value() == "NO_DESC_NO_VAL="

    def test_description_only_none_value(self):
        var = EnvVar(name="DESC_ONLY", var_type="STRING", description="Some description")
        assert var.get_dotenv_value() == "# Some description\nDESC_ONLY="

    def test_empty_description_emits_blank_comment_line(self):
        var = EnvVar(name="EMPTY_DESC", var_type="STRING", description="")
        # wrap_lines("") -> [""] so we expect "# " as a comment line
        assert var.get_dotenv_value() == "# \nEMPTY_DESC="

    def test_value_present_is_stripped_and_formatted(self):
        var = EnvVar(name="HAS_VAL", var_type="STRING", description="Desc", value="  value  ")
        # value is stripped, and formatting wraps it in quotes
        assert var.get_dotenv_value() == '# Desc\nHAS_VAL="value"'

    def test_value_with_quotes_is_escaped(self):
        var = EnvVar(name="QUOTE", var_type="STRING", description="Desc", value=' He said "hi" ')
        # value is stripped and quotes are escaped by the formatter
        assert var.get_dotenv_value() == '# Desc\nQUOTE="He said \\"hi\\""'

    def test_empty_string_value_is_kept_and_quoted(self):
        var = EnvVar(name="EMPTY", var_type="STRING", description=None, value="")
        # empty string is not None -> formatter should render KEY=""
        assert var.get_dotenv_value() == 'EMPTY=""'

    def test_multiline_description_and_empty_value_string(self):
        desc = "line1\n\nline3"
        var = EnvVar(name="MULTI", var_type="STRING", description=desc, value="")
        # wrap_lines preserves blank lines; each should be commented
        assert var.get_dotenv_value() == '# line1\n# \n# line3\nMULTI=""'
