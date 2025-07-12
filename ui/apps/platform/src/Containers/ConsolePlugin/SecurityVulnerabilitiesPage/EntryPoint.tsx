import * as React from 'react';
import { PageSection, Title } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

export function EntryPoint() {
    return (
        <>
            <PageSection>
                <Title headingLevel="h1">{'Hello, Plugin!'}</Title>
            </PageSection>
            <PageSection>
                <p>
                    <span className="console-plugin-template__nice">
                        <CheckCircleIcon /> {'Success!'}
                    </span>{' '}
                    {'Your plugin is working.'}
                </p>
                <p>
                    {
                        'This is a custom page contributed by the console plugin template. The extension that adds the page is declared in console-extensions.json in the project root along with the corresponding nav item. Update console-extensions.json to change or add extensions. Code references in console-extensions.json must have a corresponding property'
                    }
                    <code>{'exposedModules'}</code>{' '}
                    {'in package.json mapping the reference to the module.'}
                </p>
                <p>
                    {'After cloning this project, replace references to'}{' '}
                    <code>{'console-template-plugin'}</code>{' '}
                    {'and other plugin metadata in package.json with values for your plugin.'}
                </p>
            </PageSection>
        </>
    );
}
