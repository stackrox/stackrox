// This is the one place where we're allowed to import directly from 'axios'.
// All other places must use the instance exported here.
// eslint-disable-next-line no-restricted-imports
import axios from 'axios';

export default axios.create({
    timeout: 10000
});
