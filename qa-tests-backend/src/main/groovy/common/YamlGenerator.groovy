package common

import groovy.transform.CompileStatic
import io.kubernetes.client.custom.IntOrString
import org.yaml.snakeyaml.DumperOptions
import org.yaml.snakeyaml.Yaml
import org.yaml.snakeyaml.introspector.Property
import org.yaml.snakeyaml.nodes.NodeTuple
import org.yaml.snakeyaml.nodes.Tag
import org.yaml.snakeyaml.representer.Representer

@CompileStatic
class YamlGenerator {
    static String toYaml(Object object) {
        def options = new DumperOptions()
        options.setExplicitStart(true)
        options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK)
        def yaml = new Yaml(new PolicyRepresenter(), options)
        return yaml.dumpAs(object, Tag.MAP, null)
    }

    static class PolicyRepresenter extends Representer {

        PolicyRepresenter() {
            super(new DumperOptions())
        }

        @Override
        protected NodeTuple representJavaBeanProperty(Object javaBean, Property property,
                                                      Object propertyValue, Tag customTag) {
            if (propertyValue instanceof MetaClassImpl || propertyValue == null) {
                return null
            } else if (propertyValue instanceof IntOrString) {
                def pv = propertyValue as IntOrString
                return super.representJavaBeanProperty(
                        javaBean,
                        property,
                        pv.integer ?
                                pv.intValue :
                                pv.strValue,
                        customTag
                )
            }

            return super.representJavaBeanProperty(
                        javaBean,
                        property,
                        propertyValue,
                        customTag
                )
        }
    }
}
