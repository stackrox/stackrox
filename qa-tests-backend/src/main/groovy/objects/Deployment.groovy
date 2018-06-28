package objects

class Deployment {
    String name
    String image
    Map<String, String> labels = new HashMap<>()
    List<Integer> ports = new ArrayList<Integer>()

    Deployment setName(String n) {
        this.name = n
        return this
    }

    Deployment setImage(String i) {
        this.image = i
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

}
