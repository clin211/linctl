# protolint 配置文件
# 文档：https://github.com/yoheimuta/protolint
# 命令：protolint .

lint:
  rules:
    no_default: false
    add:
      - FILE_HAS_COMMENT
      - SERVICES_HAVE_COMMENT
      - RPCS_HAVE_COMMENT
      - FIELDS_HAVE_COMMENT
      - MESSAGES_HAVE_COMMENT
      - ENUMS_HAVE_COMMENT
      - ENUM_FIELDS_HAVE_COMMENT
    remove:
      - MAX_LINE_LENGTH
  rules_option:
    indent:
      style: 2
    max_line_length:
      max_chars: 120
    enum_field_names_zero_value_end_with:
      suffix: UNSPECIFIED
