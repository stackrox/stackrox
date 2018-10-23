package objects

class Pod {
    String name
    String namespace
    String uid
    String podIP
    List<String> containerIds = new ArrayList<>()

    def getPodId() {
        return "${this.name}.${this.namespace}@${this.uid}"
    }
}
