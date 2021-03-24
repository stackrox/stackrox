export type BaselineSimulationOptions = {
    excludePortsAndProtocols: boolean;
};
export type BaselineSimulationResult =
    | {
          isBaselineSimulationOn: boolean;
          baselineSimulationOptions: BaselineSimulationOptions;
          startBaselineSimulation: (baselineSimulationOptions: BaselineSimulationOptions) => void;
          stopBaselineSimulation: () => void;
      }
    | undefined;
