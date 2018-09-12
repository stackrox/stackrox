package objects

class Deployment {
    String name
    List<String> volNames = new ArrayList<String>()
    List<String> volMounts = new ArrayList<String>()
    String image
    String mountpath
    List<String> secretNames = new ArrayList<String>()
    Map<String, String> labels = new HashMap<>()
    List<Integer> ports = new ArrayList<Integer>()

    Deployment setName(String n) {
        this.name = n
        // This label will be the selector used to select this deployment.
        this.addLabel("name", n)
        return this
    }

    Deployment setImage(String i) {
        this.image = i
        return this
    }

    Deployment addMountPath(String m) {
        this.mountpath = m
        return this
    }

    Deployment addLabel(String k, String v) {
        this.labels[k] = v
        return this
    }

    Deployment addPort(Integer p) {
        this.ports.add(p)
        return this
    }

    Deployment addSecretName(String s) {
        this.secretNames.add(s)
        return this
    }

    Deployment addVolName(String v) {
        this.volNames.add(v)
        return this
    }

    Deployment addVolMountName(String v) {
        this.volMounts.add(v)
        return this
   }
}
