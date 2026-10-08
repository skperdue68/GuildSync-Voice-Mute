import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createShortcutActivation} from './shortcut-activation.js';
test('reconnect does not repeat a failed consent request; explicit retry can',async()=>{
 const activation=createShortcutActivation();let calls=0;
 const start=async()=>{calls++;throw Error('Permission declined');};
 await assert.rejects(activation.activate(start));
 assert.equal(await activation.activate(start),false);
 assert.equal(calls,1);
 activation.retry();await assert.rejects(activation.activate(start));assert.equal(calls,2);
});
