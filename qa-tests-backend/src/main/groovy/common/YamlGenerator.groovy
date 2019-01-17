package common

import io.fabric8.kubernetes.api.model.IntOrString
import org.yaml.snakeyaml.DumperOptions
import org.yaml.snakeyaml.Yaml
import org.yaml.snakeyaml.introspector.Property
import org.yaml.snakeyaml.nodes.NodeTuple
import org.yaml.snakeyaml.nodes.Tag
import org.yaml.snakeyaml.representer.Representer

class YamlGenerator {
    static String toYaml(Object object) {
        def options = new DumperOptions()
        options.setExplicitStart(true)
        options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK)
        def yaml = new Yaml(new PolicyRepresenter(), options)
        return yaml.dumpAs(object, Tag.MAP, null)
    }

    static class PolicyRepresenter extends Representer {

        @Override
        protected NodeTuple representJavaBeanProperty(Object javaBean, Property property,
                                                      Object propertyValue, Tag customTag) {
            if (propertyValue instanceof MetaClassImpl || propertyValue == null) {
                return null
            } else if (propertyValue instanceof IntOrString) {
                return super.representJavaBeanProperty(
                        javaBean,
                        property,
                        propertyValue.integer ?
                                propertyValue.intValue :
                                propertyValue.strValue,
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
