import { AdvancedFlowsFilterType } from './types';
import { filtersToSelections, selectionsToFilters } from './advancedFlowsFilterUtils';

describe('advancedFlowsFilterUtils', () => {
    describe('filtersToSelections', () => {
        it('should convert filters with flows to selections', () => {
            const filters: AdvancedFlowsFilterType = {
                flows: ['baseline', 'anomalous'],
                directionality: [],
                protocols: [],
                ports: [],
            };

            const selections = filtersToSelections(filters);

            expect(selections).toEqual(['baseline', 'anomalous']);
        });

        it('should convert filters with directionality to selections', () => {
            const filters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: ['egress', 'ingress'],
                protocols: [],
                ports: [],
            };

            const selections = filtersToSelections(filters);

            expect(selections).toEqual(['egress', 'ingress']);
        });

        it('should convert filters with protocols to selections', () => {
            const filters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: [],
                protocols: ['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP'],
                ports: [],
            };

            const selections = filtersToSelections(filters);

            expect(selections).toEqual(['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP']);
        });

        it('should convert filters with ports to selections', () => {
            const filters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: [],
                protocols: [],
                ports: ['9000', '8080'],
            };

            const selections = filtersToSelections(filters);

            expect(selections).toEqual(['9000', '8080']);
        });

        it('should convert filters with combination of values to selections', () => {
            const filters: AdvancedFlowsFilterType = {
                flows: ['anomalous', 'baseline'],
                directionality: ['egress', 'ingress'],
                protocols: ['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP'],
                ports: ['9000', '8080'],
            };

            const selections = filtersToSelections(filters);

            expect(selections).toEqual([
                'anomalous',
                'baseline',
                'egress',
                'ingress',
                'L4_PROTOCOL_TCP',
                'L4_PROTOCOL_UDP',
                '9000',
                '8080',
            ]);
        });
    });

    describe('selectionsToFilters', () => {
        it('should convert selections with flows to filters', () => {
            const selections: string[] = ['anomalous', 'baseline'];

            const filters = selectionsToFilters(selections);

            const expectedFilters: AdvancedFlowsFilterType = {
                flows: ['anomalous', 'baseline'],
                directionality: [],
                protocols: [],
                ports: [],
            };

            expect(filters).toEqual(expectedFilters);
        });

        it('should convert selections with directionality to filters', () => {
            const selections: string[] = ['ingress', 'egress'];

            const filters = selectionsToFilters(selections);

            const expectedFilters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: ['ingress', 'egress'],
                protocols: [],
                ports: [],
            };

            expect(filters).toEqual(expectedFilters);
        });

        it('should convert selections with protocols to filters', () => {
            const selections: string[] = ['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP'];

            const filters = selectionsToFilters(selections);

            const expectedFilters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: [],
                protocols: ['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP'],
                ports: [],
            };

            expect(filters).toEqual(expectedFilters);
        });

        it('should convert selections with ports to filters', () => {
            const selections: string[] = ['9000', '8080'];

            const filters = selectionsToFilters(selections);

            const expectedFilters: AdvancedFlowsFilterType = {
                flows: [],
                directionality: [],
                protocols: [],
                ports: ['9000', '8080'],
            };

            expect(filters).toEqual(expectedFilters);
        });

        it('should convert selections with combination of values to filters', () => {
            const selections: string[] = [
                'anomalous',
                'baseline',
                'egress',
                'ingress',
                'L4_PROTOCOL_TCP',
                'L4_PROTOCOL_UDP',
                '9000',
                '8080',
            ];

            const filters = selectionsToFilters(selections);

            const expectedFilters: AdvancedFlowsFilterType = {
                flows: ['anomalous', 'baseline'],
                directionality: ['egress', 'ingress'],
                protocols: ['L4_PROTOCOL_TCP', 'L4_PROTOCOL_UDP'],
                ports: ['9000', '8080'],
            };

            expect(filters).toEqual(expectedFilters);
        });
    });
});
