import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import Message from './index';

export default {
    title: 'Message',
    component: Message,
} as Meta;

export const BaseTypeWithStringContent: Story<{}> = () => (
    <Message>The most basic message imanginable.</Message>
);

export const BaseTypeWithChildren: Story<{}> = () => (
    <Message>
        <p className="mb-2">
            An existing policy with the name “Container using read-write root filesystem” has the
            same ID—8ac93446-4ad4-a275-3f518db0ceb9—as the policy “Fixable CVSS {'>'}= 9” you are
            trying to import.
        </p>
        <p>
            An existing policy has the same name, “Fixable CVSS {'>'}= 9”, as the one you are trying
            to import.
        </p>
    </Message>
);

export const SuccessType: Story<{}> = () => (
    <Message type="success">Congratulations! You won the lottery.</Message>
);

export const WarnType: Story<{}> = () => (
    <Message type="warn">
        <span>
            Central doesn&apos;t have the required Kernel support package. Retrieve it from{' '}
            <a
                href="https://install.stackrox.io/collector/support-packages/index.html"
                className="underline text-primary-900"
                target="_blank"
                rel="noopener noreferrer"
            >
                stackrox.io
            </a>{' '}
            and upload it to Central using roxctl.
        </span>
    </Message>
);

export const ErrorType: Story<{}> = () => {
    const imageScanMessages = {
        header: 'The scanner doesn’t provide OS information.',
        body:
            'Failed to get the base OS information. Either the integrated scanner can’t find the OS or the base OS is unidentifiable.',
    };
    return (
        <Message type="error">
            <div className="w-full">
                <header className="text-lg pb-2 border-b border-alert-700 mb-2 w-full">
                    <h2 className="mb-1 font-700 tracking-wide uppercase">
                        CVE Data May Be Inaccurate
                    </h2>
                    <span>{imageScanMessages.header}</span>
                </header>
                <p>{imageScanMessages.body}</p>
            </div>
        </Message>
    );
};
