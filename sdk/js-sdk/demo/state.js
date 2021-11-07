const { strict: assert } = require('assert');
const { Client } = require('layotto');

const client = new Client();
assert(client);

async function main() {
  const storeName = 'redis';
  const key = 'foo-js-sdk';
  const value = `bar, from js-sdk, ${Date()}`;

  await client.state.save(storeName, [
    { key, value },
  ]);
  console.log('saveState success, key: %j, value: %j', key, value);

  const resValue = await client.state.get(storeName, key);
  console.log('getState success, key: %j, value: %j, toString: %j', key, resValue, Buffer.from(resValue).toString('utf8'));
}

main();
