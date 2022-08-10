import React, { useEffect, useMemo, useState } from 'react';
import {
    Form,
    FormGroup,
    TextInput,
    ActionGroup,
    Button,
    PageSection,
    Title,
    SelectOption,
    Select,
    Divider,
} from '@patternfly/react-core';
import { loginWithBasicAuth } from 'services/AuthService';
import { useK8sWatchResource } from '@openshift-console/dynamic-plugin-sdk';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export default function PluginLogin({ onLogin, onEndpointChange }) {
    const { isOpen, onToggle } = useSelectToggle();
    const [endpoint, setEndpoint] = useState<string | undefined>(undefined);
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');

    // TODO Is this the most accurate way to detect Routes to central?
    // Can we make better use of `selector.matchExpressions`?
    const [routes] = useK8sWatchResource({
        groupVersionKind: {
            version: 'v1',
            kind: 'Route',
        },
        isList: true,
        namespace: 'stackrox',
        selector: {
            matchLabels: {
                'app.kubernetes.io/component': 'central',
            },
        },
    });

    const stackroxCentralRoutes = useMemo(() => {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return (routes as any[]).filter((rt) => rt.metadata.name === 'central');
    }, [routes]);

    useEffect(() => {
        if (!endpoint && stackroxCentralRoutes.length) {
            setEndpoint(stackroxCentralRoutes[0].spec.host);
        }
    }, [endpoint, stackroxCentralRoutes]);

    useEffect(() => {
        onEndpointChange(endpoint);
    }, [endpoint, onEndpointChange]);

    function login() {
        return loginWithBasicAuth(username, password, {
            id: '4df1b98c-24ed-4073-a9ad-356aec6bb62d',
            type: 'basic',
        } as any).then((res) => {
            onLogin(res);
        });
    }

    function selectHandler(e, newEndpoint) {
        setEndpoint(newEndpoint);
        onEndpointChange(newEndpoint);
    }

    return (
        <>
            <PageSection>
                <Title headingLevel="h1">Login to ACS Instance</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-u-m-lg">
                <Form>
                    <FormGroup label="Select ACS Instance" isRequired fieldId="simple-form-name-03">
                        <Select
                            isOpen={isOpen}
                            onToggle={onToggle}
                            selections={endpoint}
                            onSelect={selectHandler}
                        >
                            {stackroxCentralRoutes.map((rt) => (
                                <SelectOption key={rt.spec.host} value={rt.spec.host}>
                                    {rt.spec.host}
                                </SelectOption>
                            ))}
                        </Select>
                    </FormGroup>
                    <FormGroup label="Username" isRequired fieldId="simple-form-name-01">
                        <TextInput
                            isRequired
                            type="text"
                            id="simple-form-name-01"
                            name="simple-form-name-01"
                            aria-describedby="simple-form-name-01-helper"
                            value={username}
                            onChange={setUsername}
                        />
                    </FormGroup>
                    <FormGroup label="Password" isRequired fieldId="simple-form-email-01">
                        <TextInput
                            isRequired
                            type="password"
                            id="simple-form-email-01"
                            name="simple-form-email-01"
                            value={password}
                            onChange={setPassword}
                        />
                    </FormGroup>
                    <ActionGroup>
                        <Button variant="primary" onClick={login}>
                            Submit
                        </Button>
                    </ActionGroup>
                </Form>
            </PageSection>
        </>
    );
}
