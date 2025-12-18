import { useState } from 'react';
import {
    Bullseye,
    Button,
    Flex,
    FlexItem,
    List,
    ListItem,
    Modal,
    SearchInput,
    pluralize,
} from '@patternfly/react-core';

import useModal from 'hooks/useModal';

type KeyValue = {
    key: string;
    value: string;
};

export type KeyValueListModalProps = {
    type: string;
    keyValues: KeyValue[];
};

function filterKeyValuesBySearchValue(keyValues: KeyValue[], searchValue: string) {
    // early return of key value if the search value is empty
    const filteredKeyValues = !searchValue.trim()
        ? keyValues
        : keyValues.filter((keyValue) => {
              // enabling case-insensitive search
              return (
                  keyValue.key.toLowerCase().includes(searchValue.toLowerCase()) ||
                  keyValue.value.toLowerCase().includes(searchValue.toLowerCase())
              );
          });
    return filteredKeyValues;
}

function KeyValueListModal({ type, keyValues }: KeyValueListModalProps) {
    const { isModalOpen, openModal, closeModal } = useModal();
    const [searchValue, setSearchValue] = useState('');

    const onChange = (value: string) => {
        setSearchValue(value);
    };

    const text = pluralize(keyValues.length, type);

    const filteredKeyValues = filterKeyValuesBySearchValue(keyValues, searchValue);

    return (
        <>
            <Button variant="link" isInline onClick={openModal} isDisabled={keyValues.length === 0}>
                {text}
            </Button>
            <Modal
                variant="medium"
                title={text}
                isOpen={isModalOpen}
                onClose={closeModal}
                actions={[
                    <Button key="cancel" variant="primary" onClick={closeModal}>
                        Cancel
                    </Button>,
                ]}
            >
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <SearchInput
                            aria-label="Key value list search input"
                            placeholder={`Search by ${type}`}
                            value={searchValue}
                            onChange={(_event, value) => onChange(value)}
                            onClear={() => onChange('')}
                        />
                    </FlexItem>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        {filteredKeyValues.length === 0 && <Bullseye>No results</Bullseye>}
                        {filteredKeyValues.length !== 0 && (
                            <List isPlain isBordered className="pf-v5-u-py-sm pf-m-scrollable">
                                {filteredKeyValues.map(({ key, value }) => {
                                    const labelText = `${key}: ${value}`;
                                    return <ListItem key={labelText}>{labelText}</ListItem>;
                                })}
                            </List>
                        )}
                    </FlexItem>
                </Flex>
            </Modal>
        </>
    );
}

export default KeyValueListModal;
