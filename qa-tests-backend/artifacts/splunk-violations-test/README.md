# Splunk Violation Integrations tests artifacts

### How to update StackRox TA .spl
1. Create new `.spl` - described in [splunk-ta](https://github.com/stackrox/splunk-ta) repository
2. Replace current `.spl` file in `qa-tests-backend/artifacts/splunk-violations-test` folder with newer version
3. Update filename in `IntegrationsSplunkViolationsTest#PATH_TO_SPLUNK_TA_SPL`
4. Add file to `allowlist`

### How to update CIM TA
1. Download `.tgz` file [here](https://splunkbase.splunk.com/app/1621/)
2. Replace current `.tgz`
3. Update filename in `IntegrationsSplunkViolationsTest#PATH_TO_CIM_TA_TGZ`
4. Add file to `allowlist`
