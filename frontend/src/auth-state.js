export function createAuthCoordinator(){
 let epoch=0;
 return {invalidate(){return ++epoch;},current(){return epoch;},
  async run(read,apply,fail){const before=epoch;try{const value=await read();if(before===epoch)await apply(value);}catch(error){if(before===epoch)await fail(error);}}
 };
}
