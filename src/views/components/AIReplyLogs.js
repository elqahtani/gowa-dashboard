export default {
    name: 'AIReplyLogs',
    data() {
        return {
            loading: false,
            logs: [],
            filter: { chat_jid: '', status: '', limit: 50 },
            expanded: null,
        }
    },
    methods: {
        async openModal() {
            $('#modalAIReplyLogs').modal('show');
            await this.refresh();
        },
        async refresh() {
            this.loading = true;
            try {
                const params = new URLSearchParams();
                if (this.filter.chat_jid) params.append('chat_jid', this.filter.chat_jid);
                if (this.filter.status) params.append('status', this.filter.status);
                if (this.filter.limit) params.append('limit', this.filter.limit);
                const resp = await window.http.get('/aireply/logs?' + params.toString());
                this.logs = resp.data.results || [];
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Failed to load logs');
            } finally {
                this.loading = false;
            }
        },
        statusColor(s) {
            switch (s) {
                case 'success': return 'green';
                case 'out_of_scope': return 'yellow';
                case 'error': return 'red';
                case 'rate_limited': return 'orange';
                default: return 'grey';
            }
        },
        truncate(s, n) {
            if (!s) return '';
            return s.length > n ? s.substring(0, n) + '…' : s;
        },
        toggle(id) {
            this.expanded = this.expanded === id ? null : id;
        },
    },
    template: `
    <div class="teal card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">AI</a>
            <div class="header">AI Reply Logs</div>
            <div class="description">
                Audit log: queries, retrieved chunks, latency, tokens
            </div>
        </div>
    </div>

    <div class="ui large modal" id="modalAIReplyLogs">
        <i class="close icon"></i>
        <div class="header"><i class="clipboard list icon"></i> AI Reply Audit Log</div>
        <div class="content">
            <div class="ui form">
                <div class="three fields">
                    <div class="field">
                        <label>Chat JID</label>
                        <input type="text" v-model="filter.chat_jid" placeholder="(all)">
                    </div>
                    <div class="field">
                        <label>Status</label>
                        <select class="ui dropdown" v-model="filter.status">
                            <option value="">(all)</option>
                            <option value="success">success</option>
                            <option value="out_of_scope">out_of_scope</option>
                            <option value="error">error</option>
                            <option value="rate_limited">rate_limited</option>
                        </select>
                    </div>
                    <div class="field">
                        <label>Limit</label>
                        <input type="number" v-model.number="filter.limit" min="1" max="500">
                    </div>
                </div>
                <button class="ui primary button" :class="{loading: loading}" @click="refresh">
                    <i class="search icon"></i> Apply Filters
                </button>
            </div>
            <div class="ui divider"></div>
            <div v-if="loading" class="ui active centered inline loader"></div>
            <table v-else class="ui celled striped table">
                <thead>
                    <tr>
                        <th>Time</th><th>Chat</th><th>Status</th>
                        <th>Query</th><th>Response</th>
                        <th>Latency</th><th>Tokens (in/out)</th><th></th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-if="!logs.length"><td colspan="8">No logs yet.</td></tr>
                    <template v-for="l in logs" :key="l.id">
                        <tr>
                            <td>{{ new Date(l.created_at).toLocaleString() }}</td>
                            <td><code>{{ truncate(l.chat_jid, 22) }}</code></td>
                            <td><span :class="['ui mini', statusColor(l.status), 'label']">{{ l.status }}</span></td>
                            <td>{{ truncate(l.query, 60) }}</td>
                            <td>{{ truncate(l.response, 60) }}</td>
                            <td>{{ l.latency_ms }}ms</td>
                            <td>{{ l.tokens_in }} / {{ l.tokens_out }}</td>
                            <td><button class="ui mini button" @click="toggle(l.id)">{{ expanded === l.id ? 'Hide' : 'Detail' }}</button></td>
                        </tr>
                        <tr v-if="expanded === l.id">
                            <td colspan="8">
                                <div><b>Query:</b><pre>{{ l.query }}</pre></div>
                                <div><b>Response:</b><pre>{{ l.response }}</pre></div>
                                <div v-if="l.retrieved_chunk_ids"><b>Retrieved Chunk IDs:</b> {{ l.retrieved_chunk_ids }}</div>
                                <div v-if="l.error_message"><b>Error:</b> <span class="ui red text">{{ l.error_message }}</span></div>
                            </td>
                        </tr>
                    </template>
                </tbody>
            </table>
        </div>
    </div>
    `,
}
