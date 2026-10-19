import { createPinia } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import GlobalStore from './modules/global';
import MenuStore from './modules/menu';
import TabsStore from './modules/tabs';
import useNodeStore from './modules/node';

const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);

export { GlobalStore, MenuStore, TabsStore, useNodeStore };

export default pinia;
