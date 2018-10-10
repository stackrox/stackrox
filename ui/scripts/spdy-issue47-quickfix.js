/**
 * The workaround is taken from https://github.com/spdy-http2/spdy-transport/issues/47#issuecomment-369020176
 * It should be deleted once the original bug is fixed and released.
 */

const fs = require('fs');
const path = require('path');

const file = path.join(process.cwd(), 'node_modules/spdy-transport/lib/spdy-transport/priority.js');

const data = fs
    .readFileSync(file)
    .toString()
    .split('\n');

if (data.length < 190) {
    data.splice(73, 0, '/*');
    data.splice(75, 0, '*/');
    data.splice(
        187,
        0,
        `
	var index = utils.binarySearch(this.list, node, compareChildren);
 	this.list.splice(index, 1);
`
    );
    const text = data.join('\n');

    fs.writeFile(file, text, err => {
        if (err) console.log(err);
    });
}
