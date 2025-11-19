# Declarative configuration

## How it works

A specific filesystem location is configured to contain files describing
the configuration of a set of objects. The content of these files is parsed
and converted to internally stored objects, and loaded in the running system.

Objects that can be declaratively configured have internal traits that define
among other things whether the object was created declaratively or
using the API. Objects created declaratively should only reference objects
that are either default system objects or declaratively configured objects.

The process that does parse, convert and stores the declaratively configured
objects runs with a specific identifier in the processing context.
It should be the only process allowed to create, update or delete
declaratively configured objects. That means that API users can try to pass
objects with declarative traits to the modifying endpoints, but that the 
process will fail and return a `Not Authorized` error in that case.

### Configuration location

The configuration files are mounted configuration maps or secrets
in the central container.

The declarative configuration maintenance process watches for files under
`/run/stackrox.io/declarative-configuration`. These files are expected to
contain a multi-object YAML representation of the configured objects.

### Data flow

The basic logic of declarative configuration is to convert
structured information from the files in the watched filesystem location
to object stored in the database and in the associated business logic
object.

The high level flow is the following one:

```
  +------------------+
  |   watched file   |
  +------------------+
           |
           v
  +------------------+
  |   YAML objects   |
  +------------------+
           |
           v
  +------------------+
  |   data struct    |
  +------------------+
           |
      transformer
           |
           v
  +------------------+
  |  storage protos  |
  +------------------+
           |
        updater
           |
           v
  +------------------+
  | database objects |
  +------------------+
```

The status of the file transformation and of the updater push to the database is
stored in the database in the `declarative_config_healths` table.

## How to add an object type to the framework

### Add `Traits` to the related storage and api protobuf object types
This should be straightforward: Add a new `Traits` field of `Traits` type
in the protobuf definition of the object types to add.

### Add the new declarative type to the declarative config healths
The new declarative type should be added
to the `ResourceType` enum from the `DeclarativeConfigHealth` message
in `proto/storage/declarative_config_health.proto`

### Create conversion between storage and api protobuf objects
This step is optional but recommended for two reasons:
- The API should not directly expose the structure of the underlying storage
objects. Having this conversion should decrease the coupling between API
and storage types. 
- This allows the UI to know whether an object is maintained by
the declarative framework or by API users. It allows the UI to show the origin
of displayed objects and to prevent users from trying to modify declaratively
configured objects.
The conversion functions live in the `central/convert` subdirectory.

### Create data structures for YAML decoding
These data structures are there to allow decoding the configuration files
to process internal data structures.

The protobuf enum types require dedicated decoders. Some of the existing types
can be used as examples.

The data structures for YAML decoding live in `pkg/declarativeconfig` and
the top-level structure should implement the `Configuration` interface defined
in `pkg/declarativeconfig/configuration.go`

### Create data transformers to convert objects decoded from YAML to storage protobuf objects
The transformer is in charge of converting the decoded YAML structure
into one or more storage protobuf objects, sorted by type.
The output of the transformer is then used to feed the updaters and populate
the database.

The transformers are defined in `pkg/declarativeconfig/transform` and should
implement the `Transformer` interface defined in
`pkg/declarativeconfig/transform/transformer.go`.

### Add declarative constraints logic to the datastore layer
This step is two-fold, impacts the `Upsert` method of the datastore
and should fulfill two requirements:
- Ensure the process that pushes an object for update is allowed to do so
with regard to the origin of the upserted or deleted object.
- Ensure the object references object of allowed origin
(declaratively configured objects should not reference user-created objects).

#### Ensure the process is allowed to push the requested object
In the datastore `Upsert` method, apply the following code snipped to both
the object passed as input and the object retrieved from the database (if any):
```go
if !declarativeconfig.CanModifyResource(ctx, obj) {
	return errox.NotAuthorized.CausedByf(
		"object %q's origin is %s, cannot be modified or deleted with the current permission",
		object.GetName(), object.GetTraits().GetOrigin()
	)
}
```

### Create declarative updaters
The role of the updater is to upsert new or changed objects, as well as
to remove objects which were removed from the declarative configuration files.
The updater should implement the `ResourceUpdater` interface declared in
`central/declarativeconfig/updater/updater.go`

The `Upsert` method should be simple and straightforward, and delegate
the action to the underlying datastore.

The `DeleteResources` method should first list the items present
in the database and candidate for removal, then attempt to remove them,
and report removal errors in both its output and the declarative health
datastore.

### Register the above objects in the declarative framework
The storage protobuf types should be added to
`central/declarativeconfig/types/accepted_types.go`.
The protobuf types should also be added in the proper sequence
to the `GetSupportedProtobufTypesInProcessingOrder` function.

The newly added type should be associated
to a `DeclarativeConfigHealth_ResourceType` enum value
in `protoMessageToHealthResourceTypeMap`
in file `central/declarativeconfig/utils/health.go`.

If the storage protobuf type does not expose the `GetId` and `GetName` methods,
alternatives should be added to the `UniversalIDExtractor`
and `UniversalNameExtractor` information extractors in
`central/declarativeconfig/types/extractors.go`.

The newly created `ResourceUpdater` should be added
to the `DefaultResourceUpdaters` function
in `central/declarativeconfig/updater/updater.go`.

The created `Configuration` type for YAML conversion should be listed
in the `getEmptyConfigurations` function
in file `pkg/declarativeconfig/configuration.go`.

The created `Transformer` should be referenced in the `New` function
in file `pkg/declarativeconfig/transform/transformer.go`.

### Ensure the UI displays object origin and prevents changes to declarative objects
The type obtained from the backend should be extended to expose traits.
This will allow alterations to the list pages as well as display/edit forms
and prevent the users from being allowed to call delete or modify on the object
from the UI.

Adding traits to the object type retrieved from the backend can usually be done
in `ui/apps/platform/src/services`.

For integration types, the `originColumnDescriptor,` column descriptor
can be added for the added object type in file
`ui/apps/platform/src/Containers/Integrations/utils/tableColumnDescriptor.ts`

For integration types, the `hasTraitsLabel` condition should be updated in file
`ui/apps/platform/src/Containers/Integrations/IntegrationPage.tsx`.

The next subsections provide rules of thumb on usage of object traits
in UI pages.

#### Add TraitsOriginLabel to the object display/edit form 
```typescript
// The imported file below is located in ui/apps/platform/src/Containers
import { TraitsOriginLabel } from '../TraitsOriginLabel';

// ...

function SomeObjectForm(...: XXXFormProps): ReactElement {
    // Initialization code
    return (
        <Form>
            <Toolbar>
                <ToolbarItem>
                    <Title headingLevel="h1">{formTitle}</Title>
                </ToolbarItem>
                // Display object origin on the form page
                {action !== 'create' && (
                    <ToolbarItem>
                        <TraitsOriginLabel traits={selectedObject.traits} />
                    </ToolbarItem>
                )}
                // Next is the 'Save' Button, conditioned by 'isActionable'
            </Toolbar>
        </Form>
    );
}
```

#### Add Tooltip information about declarative and system objects not being editable in the display form
```typescript
// The imported file below is located in ui/apps/platform/src/Containers
import { isUserResource } from '../traits';

// ...

function SomeObjectForm(...: XXXFormProps): ReactElement {
    // Initialization code
    return (
        <Form>
            <!-- Previous form objects -->
            <ParentFormObjects>
                {!isUserResource(selectedObject.traits) && (
                    <FlexItem>
                        <Tooltip content="Object is managed declaratively and can only be edited declaratively.">
                            <Button
                                variant="plain"
                                aria-label="Information button"
                                style={{
                                    transform:
                                        'translate(0, 42px)',
                                }}
                            >
                                <InfoCircleIcon />
                            </Button>
                        </Tooltip>
                    </FlexItem>
                )}
            </ParentFormObjects>
        </Form>
    );
}
```

#### Add origin label to list pages

```typescript
// The imported file below is located in ui/apps/platform/src/Containers
import { getOriginLabel } from '../traits';

// ...

function SomeObjectsList(...: XXXFormProps): ReactElement {
    // Initialization code
    return (
        <>
            <!-- Previous form objects -->
            <Table>
                <Thead>
                    <Tr>
                        <Th width={15}>Name</Th>
                        <Th width={15}>Origin</Th>
                        <!-- Other table headers -->
                    </Tr>
                </Thead>
                <TBody>
                    {objects.map(({ id, name, xxx, traits }) => (
                        <Tr key={id}>
                            <Td dataLabel="Name">{name}</Td>
                            <Td dataLabel="Origin">{getOriginLabel(traits)}</Td>
                            <!-- Other row cells -->
                        </Tr>
                    ))}
                </TBody>
            </Table>
        </>
    );
}
```

