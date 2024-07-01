FROM nginx:1.27.0

ADD trigger-violations.sh /trigger-violations-insecure.sh

LABEL test.com-i-am-a-key="a value" test.com-i-am-another-key="another value"

CMD ["sh", "-x", "trigger-violations-insecure.sh"]
