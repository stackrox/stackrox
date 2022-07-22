// The main configuration storage type. Reading directly from localStorage would provide this type
// of result at the top level.
export type WidgetConfigStorage = Record<WidgetId, WidgetConfigMap>;

// A unique identifier for a widget. Short term this will be the name of the widget, but if
// we move to server side storage this could become a UUID.
export type WidgetId = string;

// A configuration object that maps routes to configurations for a specific widget.
type WidgetConfigMap = Partial<Record<RouteId, WidgetConfig>>;

// A path identifier in the application. This should be a route that maps to a route in the app, with a hash
// parameter used to reference individual sections of a page.
//
// e.g. /main/dashboard
//      /main/dashboard/trends-tab
//      /main/dashboard/trends-tab#modal
//      /main/dashboard/trends-tab#sidebar
export type RouteId = string;

// Configuration read for a single instance of a widget.
export type WidgetConfig = Readonly<Partial<Record<string, ConfigOptionValue>>>;

// A widget config is an object that can contain the following properties
type ConfigOptionValue = OneOfValue | AnyOfValue | ToggleValue | Readonly<ToggleValue[]>;
// A simple, single string value. Used when the user can select exactly one option from a selection.
type OneOfValue = string | number;
// An array of selected options.
type AnyOfValue = readonly string[] | readonly number[];
// An editable value that can be toggled on or off.
type ToggleValue = Readonly<{ enabled: boolean; value: number | string | boolean }>;
