// A failed consent request is retried only after an explicit user action.
export function createShortcutActivation() {
 let failed=false,pending=false;
 return {
  retry(){failed=false;},
  async activate(start){if(failed||pending)return false;pending=true;try{await start();return true;}catch(error){failed=true;throw error;}finally{pending=false;}}
 };
}
