function getDirectionalityLabel(isIngress: boolean): string {
    return isIngress ? 'Ingress' : 'Egress';
}

export default getDirectionalityLabel;
