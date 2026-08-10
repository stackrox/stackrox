import { useCallback, useState } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import {
    Alert,
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import { MinusCircleIcon, PlusCircleIcon } from '@patternfly/react-icons';
import { useField } from 'formik';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { integrationsPath } from 'routePaths';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { listCollections } from 'services/CollectionsService';
import type { CollectionSlim } from 'services/CollectionsService';
import { getTableUIState } from 'utils/getTableUIState';
import type { NotifierCollectionBinding } from 'types/policy.proto';

import {
    buildNotificationRoutes,
    hasDuplicateRoute,
    splitNotificationRoutes,
} from '../../policies.utils';
import type { NotificationRoute } from '../../policies.utils';

const ALL_SCOPES = '';

function NotifiersForm() {
    const [notifiersField, , notifiersHelpers] = useField<string[]>('notifiers');
    const [mappingsField, , mappingsHelpers] = useField<NotifierCollectionBinding[]>(
        'notifierToCollectionMappings'
    );

    const routes = buildNotificationRoutes(notifiersField.value, mappingsField.value);

    const [isAddingRoute, setIsAddingRoute] = useState(false);
    const [newNotifierId, setNewNotifierId] = useState('');
    const [newCollectionId, setNewCollectionId] = useState(ALL_SCOPES);
    const [isNotifierSelectOpen, setIsNotifierSelectOpen] = useState(false);
    const [isCollectionSelectOpen, setIsCollectionSelectOpen] = useState(false);
    const [duplicateError, setDuplicateError] = useState('');

    const fetchNotifiers = useCallback(() => fetchNotifierIntegrations(), []);
    const { data: notifiers = [], isLoading: isLoadingNotifiers, error: notifiersError } =
        useRestQuery(fetchNotifiers);

    const fetchCollections = useCallback(
        () => listCollections({}, { field: 'Collection Name', reversed: false }, 0, 0),
        []
    );
    const { data: collectionsRaw = [] } = useRestQuery(fetchCollections);
    const collections: CollectionSlim[] = collectionsRaw.map((c) => ({
        id: c.id,
        name: c.name,
        description: c.description,
    }));

    const tableState = getTableUIState({
        isLoading: isLoadingNotifiers,
        data: routes,
        error: notifiersError,
        searchFilter: {},
    });

    function getNotifierName(notifierId: string): string {
        return notifiers.find((n) => n.id === notifierId)?.name ?? notifierId;
    }

    function getNotifierType(notifierId: string): string {
        return notifiers.find((n) => n.id === notifierId)?.type ?? '';
    }

    function getCollectionName(collectionId: string): string {
        if (!collectionId) {
            return 'All scopes';
        }
        return collections.find((c) => c.id === collectionId)?.name ?? collectionId;
    }

    function applyRoutes(updatedRoutes: NotificationRoute[]) {
        const { notifiers: updatedNotifiers, notifierToCollectionMappings } =
            splitNotificationRoutes(updatedRoutes);
        notifiersHelpers.setValue(updatedNotifiers);
        mappingsHelpers.setValue(notifierToCollectionMappings);
    }

    function handleRemoveRoute(index: number) {
        const updatedRoutes = routes.filter((_, i) => i !== index);
        applyRoutes(updatedRoutes);
    }

    function handleAddRoute() {
        if (!newNotifierId) {
            return;
        }

        const collectionName = newCollectionId
            ? collections.find((c) => c.id === newCollectionId)?.name ?? ''
            : '';
        const newRoute: NotificationRoute = {
            notifierId: newNotifierId,
            collectionId: newCollectionId,
            collectionName,
        };

        if (hasDuplicateRoute(routes, newRoute)) {
            setDuplicateError('This notifier and scope combination already exists.');
            return;
        }

        applyRoutes([...routes, newRoute]);
        setNewNotifierId('');
        setNewCollectionId(ALL_SCOPES);
        setIsAddingRoute(false);
        setDuplicateError('');
    }

    function handleCancelAdd() {
        setIsAddingRoute(false);
        setNewNotifierId('');
        setNewCollectionId(ALL_SCOPES);
        setDuplicateError('');
    }

    return (
        <Form>
            <Table borders>
                <Thead>
                    <Tr>
                        <Th>Notifier</Th>
                        <Th>Type</Th>
                        <Th>Scope</Th>
                        <Th />
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={4}
                    errorProps={{
                        title: 'There was an error loading notifiers',
                    }}
                    emptyProps={{
                        message:
                            'No notification routes configured. Add a route to forward policy violations to a notifier.',
                        children: (
                            <Link to={integrationsPath} target="_blank" rel="noopener noreferrer">
                                Go to integrations
                            </Link>
                        ),
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((route, rowIndex) => (
                                <Tr key={`${route.notifierId}-${route.collectionId}`}>
                                    <Td dataLabel="Notifier">
                                        {getNotifierName(route.notifierId)}
                                    </Td>
                                    <Td dataLabel="Type">{getNotifierType(route.notifierId)}</Td>
                                    <Td dataLabel="Scope">
                                        {getCollectionName(route.collectionId)}
                                    </Td>
                                    <Td isActionCell>
                                        <Button
                                            variant="plain"
                                            aria-label="Remove notification route"
                                            onClick={() => handleRemoveRoute(rowIndex)}
                                        >
                                            <MinusCircleIcon />
                                        </Button>
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    )}
                />
            </Table>

            {isAddingRoute && (
                <Flex
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                    className="pf-v5-u-mt-md pf-v5-u-p-md"
                    style={{ border: '1px solid var(--pf-v5-global--BorderColor--100)' }}
                >
                    <FormGroup label="Notifier" isRequired fieldId="new-route-notifier">
                        <Select
                            id="new-route-notifier"
                            isOpen={isNotifierSelectOpen}
                            onOpenChange={setIsNotifierSelectOpen}
                            toggle={(toggleRef) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    onClick={() => setIsNotifierSelectOpen(!isNotifierSelectOpen)}
                                    isExpanded={isNotifierSelectOpen}
                                    style={{ width: '100%' }}
                                >
                                    {newNotifierId
                                        ? getNotifierName(newNotifierId)
                                        : 'Select a notifier'}
                                </MenuToggle>
                            )}
                            onSelect={(_event, value) => {
                                setNewNotifierId(value as string);
                                setIsNotifierSelectOpen(false);
                                setDuplicateError('');
                            }}
                            selected={newNotifierId}
                        >
                            <SelectList>
                                {notifiers.map((notifier) => (
                                    <SelectOption key={notifier.id} value={notifier.id}>
                                        {notifier.name} ({notifier.type})
                                    </SelectOption>
                                ))}
                            </SelectList>
                        </Select>
                        {notifiers.length === 0 && (
                            <FormHelperText>
                                <HelperText>
                                    <HelperTextItem variant="warning">
                                        No notifiers available.{' '}
                                        <Link
                                            to={integrationsPath}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            Configure integrations
                                        </Link>{' '}
                                        first.
                                    </HelperTextItem>
                                </HelperText>
                            </FormHelperText>
                        )}
                    </FormGroup>

                    <FormGroup label="Scope" fieldId="new-route-collection">
                        <Select
                            id="new-route-collection"
                            isOpen={isCollectionSelectOpen}
                            onOpenChange={setIsCollectionSelectOpen}
                            toggle={(toggleRef) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    onClick={() =>
                                        setIsCollectionSelectOpen(!isCollectionSelectOpen)
                                    }
                                    isExpanded={isCollectionSelectOpen}
                                    style={{ width: '100%' }}
                                >
                                    {getCollectionName(newCollectionId)}
                                </MenuToggle>
                            )}
                            onSelect={(_event, value) => {
                                setNewCollectionId(value as string);
                                setIsCollectionSelectOpen(false);
                                setDuplicateError('');
                            }}
                            selected={newCollectionId}
                        >
                            <SelectList>
                                <SelectOption value="">All scopes</SelectOption>
                                {collections.map((collection) => (
                                    <SelectOption key={collection.id} value={collection.id}>
                                        {collection.name}
                                    </SelectOption>
                                ))}
                            </SelectList>
                        </Select>
                    </FormGroup>

                    {duplicateError && (
                        <Alert variant="warning" isInline isPlain title={duplicateError} />
                    )}

                    <Flex>
                        <FlexItem>
                            <Button
                                variant="primary"
                                isDisabled={!newNotifierId}
                                onClick={handleAddRoute}
                            >
                                Add
                            </Button>
                        </FlexItem>
                        <FlexItem>
                            <Button variant="link" onClick={handleCancelAdd}>
                                Cancel
                            </Button>
                        </FlexItem>
                    </Flex>
                </Flex>
            )}

            {!isAddingRoute && (
                <Button
                    variant="link"
                    icon={<PlusCircleIcon />}
                    className="pf-v5-u-mt-sm"
                    onClick={() => setIsAddingRoute(true)}
                >
                    Add notification route
                </Button>
            )}
        </Form>
    );
}

export default NotifiersForm;
