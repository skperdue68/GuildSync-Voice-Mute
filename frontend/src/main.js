import {io} from 'socket.io-client';
import {Events} from '@wailsio/runtime';
import * as backend from '../bindings/github.com/skperdue68/GuildSync-Voice-Mute/app.js';
import {LoginComplete,LoginError,ShortcutError,VoiceShortcutEdge,VoiceShortcutStatus} from '../bindings/events/index.js';
import {createVoiceSession} from './voice-session.js';
import {createShortcutCapture} from './shortcut-capture.js';
import {createAuthCoordinator,canEnableVoiceSession} from './auth-state.js';
import './style.css';
import {createShortcutActivation} from './shortcut-activation.js';
const activation=createShortcutActivation();
let availability={available:true,reason:''};
const api=()=>backend;
const auth=createAuthCoordinator();
let session=null,socket=null,settings={enabled:false,shortcut:'Ctrl+M',timeout_seconds:10},capturing=false,busy=false,startAtLogin=false,startAtLoginBlocked='',updateInfo=null;
document.querySelector('#app').innerHTML=`<h1>GuildSync Voice Mute</h1><p id="appVersion" class="hint"></p><p class="intro">Press your shortcut to mute lower-ranked members in your Discord voice channel. The mute ends on its own, and pressing again restarts the countdown.</p><section><p id="account">Not signed in</p><div class="buttons"><button id="login">Sign in with Discord</button><button id="logout" hidden>Sign out</button></div></section><section><label class="toggle"><input type="checkbox" id="enabled"> Enable the mute shortcut</label><p class="hint" id="enabledHint">Works in any app while the companion is running.</p><p>Shortcut: <strong id="shortcut">Ctrl+M</strong></p><p><label for="timeout">Auto-unmute after <input type="number" id="timeout" min="3" max="120" step="1" value="10"> seconds</label></p><div class="buttons"><button id="capture">Set shortcut</button><button id="default" class="secondary">Return to Default</button></div><p class="hint" id="captureHelp">Press one key with optional Ctrl, Alt or Shift, then release it to save. Escape cancels.</p><label class="toggle"><input type="checkbox" id="startAtLogin"> Start at login</label><p class="hint" id="startAtLoginHint" hidden></p></section><section><div class="buttons"><button id="checkUpdates">Check for updates</button><button id="releasePage" class="secondary" hidden>Open download page</button></div><p id="updateStatus" class="hint" role="status"></p></section><p id="availability" role="status"></p><p id="status" role="status" aria-live="polite">Starting…</p><p class="hint">Moderator mutes are preserved. Equal and higher ranks are protected. The mute ends automatically after the configured time. Closing the window keeps the companion running in the system tray when your desktop has one.</p>`;
const el=id=>document.getElementById(id);
function status(message){el('status').textContent=String(message||'');}
function render(){el('availability').textContent=availability.reason;el('account').textContent=session?.logged_in?`Signed in as ${session.user.display_name||session.user.username}`:'Not signed in';el('login').hidden=Boolean(session?.logged_in);el('logout').hidden=!session?.logged_in;el('shortcut').textContent=settings.shortcut;el('enabled').checked=settings.enabled;el('timeout').value=settings.timeout_seconds;el('timeout').disabled=busy;el('startAtLogin').checked=startAtLogin;el('startAtLogin').disabled=busy||Boolean(startAtLoginBlocked);el('startAtLoginHint').hidden=!startAtLoginBlocked;el('startAtLoginHint').textContent=startAtLoginBlocked;el('enabled').disabled=busy||!availability.available||!canEnableVoiceSession(session);el('enabledHint').textContent=!session?.logged_in?'Sign in with Discord to enable the mute shortcut.':!canEnableVoiceSession(session)?'This account is not approved for voice mutes.':'Works in any app while the companion is running.';el('capture').disabled=busy;el('default').disabled=busy;el('checkUpdates').disabled=busy;}
const controller=createVoiceSession({getSocket:()=>socket,notify:status});
async function disableListener(){activation.invalidate();controller.stop();await api().SetShortcutActive(false);}
function canActivate(){return Boolean(socket?.connected&&settings.enabled&&!capturing&&availability.available);}
async function activate(){if(!canActivate())return;try{if(await activation.activate(()=>api().SetShortcutActive(true),canActivate))status('Ready. Press your shortcut while in a Discord voice channel.');}catch(error){availability=await api().GetShortcutAvailability();render();status(availability.reason||error);}}
function clearSocket(){controller.stop();socket?.disconnect();socket=null;}
async function connect(s){const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=s;render();if(!s?.allowed||!s.token){status('Sign in with Discord. Your server role determines mute access.');return;}socket=io(s.socket_url,{auth:{source:'voice-mute',token:s.token}});socket.on('connect',()=>{activation.invalidate();void activate();});socket.on('disconnect',()=>{activation.invalidate();controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Disconnected. Reconnecting…');});socket.on('connect_error',()=>{activation.invalidate();controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Connection unavailable. Check that your Discord login is current and the server is online.');});}
async function save(next){activation.retry();busy=true;render();try{await disableListener();await api().SetShortcutSettings(next);settings=await api().GetShortcutSettings();await activate();}catch(error){status(error);}finally{busy=false;render();}}
el('login').onclick=async()=>{auth.invalidate();try{await api().StartDiscordLogin();status('Complete Discord login in your browser.');}catch(error){status(error);}};
el('logout').onclick=async()=>{auth.invalidate();try{await disableListener();clearSocket();await api().Logout();session=null;render();status('Signed out.');}catch(error){status(error);}};
el('enabled').onchange=()=>save({...settings,enabled:el('enabled').checked});
el('default').onclick=()=>save({enabled:false,shortcut:'Ctrl+M',timeout_seconds:10});
el('timeout').onchange=()=>save({...settings,timeout_seconds:Number(el('timeout').value)||10});
function startAtLoginBlockedReason(){return /Mac/i.test(navigator.platform||navigator.userAgent)?'macOS has not approved the login item yet. Open System Settings → General → Login Items (Login Items & Extensions on macOS 15 or later), allow GuildSync Voice Mute, then reopen the app.':'The desktop did not keep the login entry. Check your startup applications for GuildSync Voice Mute and try again.';}
el('startAtLogin').onchange=async()=>{const wanted=el('startAtLogin').checked;busy=true;render();try{await api().SetStartAtLogin(wanted);startAtLogin=await api().GetStartAtLogin();if(startAtLogin!==wanted){startAtLoginBlocked=startAtLoginBlockedReason();status(startAtLoginBlocked);}else{startAtLoginBlocked='';status(startAtLogin?'The companion will start with your computer.':'The companion will no longer start automatically.');}}catch(error){startAtLogin=await api().GetStartAtLogin();startAtLoginBlocked=String(error||'Starting at login could not be changed on this computer.');status(startAtLoginBlocked);}finally{busy=false;render();}};
el('checkUpdates').onclick=async()=>{busy=true;render();el('updateStatus').textContent='Checking for updates…';try{updateInfo=await api().CheckForUpdate();el('updateStatus').textContent=updateInfo.message;el('releasePage').hidden=!updateInfo.available;}catch(error){el('updateStatus').textContent=String(error);}finally{busy=false;render();}};
el('releasePage').onclick=async()=>{try{await api().OpenReleasePage(updateInfo.page_url);}catch(error){el('updateStatus').textContent=String(error);}};
el('capture').onclick=async()=>{try{await disableListener();capturing=true;shortcutCapture.start();el('captureHelp').textContent='Press one key with optional Ctrl, Alt or Shift; release it to save. Escape cancels.';el('capture').textContent='Listening…';}catch(error){status(error);}};
function endCapture(){capturing=false;el('capture').textContent='Set shortcut';el('captureHelp').textContent='Press one key with optional Ctrl, Alt or Shift, then release it to save. Escape cancels.';}
const shortcutCapture=createShortcutCapture({
 save:shortcut=>{endCapture();void save({...settings,shortcut});},
 cancel:()=>{endCapture();void activate();},error:status,
 changed:shortcut=>{el('captureHelp').textContent=`${shortcut} — release to save`;}
});
window.addEventListener('keydown',event=>shortcutCapture.keydown(event));
window.addEventListener('keyup',event=>shortcutCapture.keyup(event));
window.addEventListener('blur',()=>shortcutCapture.cancel());
Events.On(VoiceShortcutEdge,event=>controller.onEdge(event.data));
Events.On(VoiceShortcutStatus,event=>status(event.data));
Events.On(LoginComplete,event=>connect(event.data).catch(status));
Events.On(LoginError,event=>status(event.data));
Events.On(ShortcutError,event=>{activation.fail();controller.stop();api().SetShortcutActive(false).catch(()=>{});status(event.data);});
window.addEventListener('beforeunload',()=>{controller.stop();socket?.disconnect();});
// Revalidate saved accounts while running. Socket-side checks remain authoritative for every request.
let verifying=false;
setInterval(async()=>{if(!session?.logged_in||verifying)return;verifying=true;try{await auth.run(()=>api().GetSession(),fresh=>{if(!fresh.logged_in)throw Error('Sign in again.');session=fresh;render();},async error=>{const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=null;render();status(error);});}finally{verifying=false;}},30000);
async function initialize(){try{el('appVersion').textContent='Version '+await api().GetAppVersion();if(!await api().StartAtLoginSupported())startAtLoginBlocked='This build cannot register a login item, so the option stays off.';startAtLogin=await api().GetStartAtLogin();settings=await api().GetShortcutSettings();await connect(await api().GetSession());}catch(error){render();status(error);}}
initialize();
