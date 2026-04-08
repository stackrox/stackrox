import {
    Alert,
    Button,
    List,
    ListItem,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
} from '@patternfly/react-core';

export type ErrorsForGeneratedPolicy = {
    alteredSearchTerms: string[];
    errorFromCatch: string;
    hasNestedFields: boolean;
};

export type GeneratedPolicyErrorModalProps = {
    errors: ErrorsForGeneratedPolicy;
    onClose: () => void;
};

function GeneratedPolicyErrorModal({ errors, onClose }: GeneratedPolicyErrorModalProps) {
    const { alteredSearchTerms, errorFromCatch, hasNestedFields } = errors;

    return (
        <Modal isOpen onClose={onClose} variant="small">
            <ModalHeader title="Generated policy" />
            <ModalBody>
                {errorFromCatch ? (
                    <Alert
                        variant="danger"
                        isInline
                        component="p"
                        title="Generated policy has errors"
                    >
                        {errorFromCatch}
                    </Alert>
                ) : (
                    <Alert
                        variant="warning"
                        isInline
                        component="p"
                        title="Generated policy has errors"
                    >
                        {hasNestedFields && <p>Policy contained nested fields.</p>}
                        {alteredSearchTerms.length !== 0 && (
                            <>
                                <p>The following search terms were removed or altered:</p>
                                <List>
                                    {alteredSearchTerms.map((alteredSearchTerm) => (
                                        <ListItem key={alteredSearchTerm}>
                                            {alteredSearchTerm}
                                        </ListItem>
                                    ))}
                                </List>
                            </>
                        )}
                    </Alert>
                )}
            </ModalBody>
            <ModalFooter>
                <Button variant="primary" onClick={onClose}>
                    Close
                </Button>
            </ModalFooter>
        </Modal>
    );
}

export default GeneratedPolicyErrorModal;
