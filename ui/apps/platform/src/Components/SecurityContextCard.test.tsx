import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';

import type { Container } from 'types/deployment.proto';
import SecurityContextCard from './SecurityContextCard';

const mockContainerWithSecurityContext = {
    id: 'container-1',
    securityContext: {
        privileged: true,
        addCapabilities: ['NET_ADMIN', 'SYS_TIME'],
        dropCapabilities: ['MKNOD'],
    },
} as unknown as Container;

const mockContainerWithoutSecurityContext = {
    id: 'container-2',
    securityContext: {
        privileged: false,
        addCapabilities: [],
        dropCapabilities: [],
    },
} as unknown as Container;

describe('SecurityContextCard', () => {
    it('filters out containers without meaningful security context', () => {
        const containers = [mockContainerWithoutSecurityContext, mockContainerWithSecurityContext];
        render(<SecurityContextCard containers={containers} />);

        expect(screen.getByText('Privileged')).toBeInTheDocument();
        expect(screen.queryByText('test-container-2')).not.toBeInTheDocument();
    });
});
