import { Component } from 'react';
import type { ComponentProps, ComponentType, ElementType } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

const componentNotMounted = (
    <Bullseye>
        <Spinner />
    </Bullseye>
);

type State = {
    component: ElementType | null;
};

export default function asyncComponent<T extends ElementType>(
    importComponent: () => Promise<{ default: T }>
): ComponentType<ComponentProps<T>> {
    class AsyncComponent extends Component<ComponentProps<T>, State> {
        isComponentMounted: boolean;

        constructor(props: ComponentProps<T>) {
            super(props);
            this.state = {
                component: null,
            };
            this.isComponentMounted = false;
        }

        async componentDidMount() {
            this.isComponentMounted = true;
            const { default: component } = await importComponent();
            if (this.isComponentMounted) {
                this.setState({ component });
            }
        }

        componentWillUnmount() {
            this.isComponentMounted = false;
        }

        render() {
            const C = this.state.component;
            return C ? <C {...this.props} /> : componentNotMounted;
        }
    }

    return AsyncComponent;
}
