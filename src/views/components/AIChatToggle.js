export default {
    name: 'AIChatToggle',
    data() {
        return {
            loading: false,
            saving: false,
            settings: [],
            newJid: '',
            newEnabled: true,
        }
    },
    methods: {
        async openModal() {
            $('#modalAIChatToggle').modal('show');
            await this.refresh();
        },
        async refresh() {
            this.loading = true;
            try {
                const resp = await window.http.get('/aireply/chat-settings');
                this.settings = resp.data.results || [];
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Failed to load');
            } finally {
                this.loading = false;
            }
        },
        async add() {
            const jid = this.newJid.trim();
            if (!jid) { showErrorInfo('Enter chat JID'); return; }
            await this.setEnabled(jid, this.newEnabled);
            this.newJid = '';
            await this.refresh();
        },
        async toggle(s) {
            await this.setEnabled(s.chat_jid, !s.enabled);
            await this.refresh();
        },
        async setEnabled(jid, enabled) {
            this.saving = true;
            try {
                await window.http.put('/aireply/chat-settings/' + encodeURIComponent(jid), { enabled });
                showSuccessInfo(enabled ? 'Enabled AI for ' + jid : 'Disabled AI for ' + jid);
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Failed to save');
            } finally {
                this.saving = false;
            }
        },
    },
    template: `
    <div class="teal card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">AI</a>
            <div class="header">AI Chat Toggle</div>
            <div class="description">
                Enable AI auto-reply per chat (opt-in, default off)
            </div>
        </div>
    </div>

    <div class="ui large modal" id="modalAIChatToggle">
        <i class="close icon"></i>
        <div class="header"><i class="comment alternate icon"></i> Per-Chat AI Toggles</div>
        <div class="content">
            <div class="ui form">
                <div class="two fields">
                    <div class="field">
                        <label>Add chat JID</label>
                        <input type="text" v-model="newJid" placeholder="62812xxxx@s.whatsapp.net">
                    </div>
                    <div class="field">
                        <label>&nbsp;</label>
                        <div class="ui toggle checkbox">
                            <input type="checkbox" v-model="newEnabled">
                            <label>Enabled</label>
                        </div>
                    </div>
                </div>
                <button class="ui primary button" :class="{loading: saving}" @click="add">
                    <i class="plus icon"></i> Save
                </button>
                <button class="ui button" @click="refresh"><i class="refresh icon"></i> Refresh</button>
            </div>
            <div class="ui divider"></div>
            <div v-if="loading" class="ui active centered inline loader"></div>
            <table v-else class="ui celled table">
                <thead><tr><th>Chat JID</th><th>Enabled</th><th>Updated</th><th></th></tr></thead>
                <tbody>
                    <tr v-if="!settings.length"><td colspan="4">No per-chat settings yet.</td></tr>
                    <tr v-for="s in settings" :key="s.chat_jid">
                        <td><code>{{ s.chat_jid }}</code></td>
                        <td>
                            <span :class="['ui', s.enabled ? 'green' : 'grey', 'label']">
                                {{ s.enabled ? 'ON' : 'OFF' }}
                            </span>
                        </td>
                        <td>{{ new Date(s.updated_at).toLocaleString() }}</td>
                        <td>
                            <button class="ui mini button" @click="toggle(s)">
                                {{ s.enabled ? 'Disable' : 'Enable' }}
                            </button>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
    `,
}
