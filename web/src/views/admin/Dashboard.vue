<template>
  <div class="admin-dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #409EFF;">
              <el-icon><UserFilled /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.user_count || 0 }}</div>
              <div class="stat-label">总用户数</div>
              <div class="stat-change positive">+{{ stats.user_growth || 0 }} 今日</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #67C23A;">
              <el-icon><Link /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.channel_count || 0 }}</div>
              <div class="stat-label">总渠道数</div>
              <div class="stat-change positive">+{{ stats.channel_growth || 0 }} 今日</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #E6A23C;">
              <el-icon><RefreshLeft /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.today_requests || 0 }}</div>
              <div class="stat-label">今日请求</div>
              <div class="stat-change" :class="stats.request_growth >= 0 ? 'positive' : 'negative'">
                {{ stats.request_growth >= 0 ? '+' : '' }}{{ stats.request_growth || 0 }} 较昨日
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #F56C6C;">
              <el-icon><DataAnalysis /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_tokens || 0 }}</div>
              <div class="stat-label">总 Token 消耗</div>
              <div class="stat-change positive">+{{ stats.token_growth || 0 }} 今日</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #909399;">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_tokens || 0 }}</div>
              <div class="stat-label">总 Token 消耗</div>
              <div class="stat-change positive">+{{ stats.token_growth || 0 }} 今日</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #409EFF;">
              <el-icon><Coin /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">¥{{ stats.total_revenue || 0 }}</div>
              <div class="stat-label">总收入</div>
              <div class="stat-change positive">+¥{{ stats.revenue_growth || 0 }} 今日</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #67C23A;">
              <el-icon><Check /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.success_rate || 0 }}%</div>
              <div class="stat-label">请求成功率</div>
              <div class="stat-change" :class="stats.success_rate_change >= 0 ? 'positive' : 'negative'">
                {{ stats.success_rate_change >= 0 ? '+' : '' }}{{ stats.success_rate_change || 0 }}% 较昨日
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #E6A23C;">
              <el-icon><Clock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.avg_response_time || 0 }}ms</div>
              <div class="stat-label">平均响应时间</div>
              <div class="stat-change" :class="stats.avg_response_time_change >= 0 ? 'negative' : 'positive'">
                {{ stats.avg_response_time_change >= 0 ? '+' : '' }}{{ stats.avg_response_time_change || 0 }}ms 较昨日
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>请求趋势</span>
              <el-select v-model="timeRange" placeholder="时间范围" size="small" @change="loadChartData">
                <el-option label="今日" value="today" />
                <el-option label="近7天" value="7days" />
                <el-option label="近30天" value="30days" />
                <el-option label="近90天" value="90days" />
              </el-select>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!chartData.length" description="暂无数据" />
            <v-chart v-else :option="requestChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>模型使用分布</span>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!modelData.length" description="暂无数据" />
            <v-chart v-else :option="modelChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>渠道状态</span>
              <el-button type="primary" size="small" @click="refreshChannelData">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!channelData.length" description="暂无数据" />
            <v-chart v-else :option="channelChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>收入统计</span>
              <el-select v-model="revenueTimeRange" placeholder="时间范围" size="small" @change="loadRevenueData">
                <el-option label="近7天" value="7days" />
                <el-option label="近30天" value="30days" />
                <el-option label="近90天" value="90days" />
              </el-select>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!revenueData.length" description="暂无数据" />
            <v-chart v-else :option="revenueChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>用户增长趋势</span>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!userGrowthData.length" description="暂无数据" />
            <v-chart v-else :option="userGrowthChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>Token 消耗趋势</span>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!tokenConsumptionData.length" description="暂无数据" />
            <v-chart v-else :option="tokenConsumptionChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近请求</span>
              <el-button type="primary" size="small" @click="refreshRequests">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          <el-table :data="recentRequests" style="width: 100%" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_email" label="用户" />
            <el-table-column prop="model_name" label="模型" />
            <el-table-column prop="path" label="路径" />
            <el-table-column prop="status_code" label="状态码" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status_code === 200 ? 'success' : 'danger'">
                  {{ scope.row.status_code }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="响应时间(ms)" width="120" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import api from '@/api'
import { Check, Clock, Coin, DataAnalysis, Key, Link, Refresh, RefreshLeft, UserFilled } from '@element-plus/icons-vue'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import {
GridComponent,
LegendComponent,
TitleComponent,
ToolboxComponent,
TooltipComponent
} from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { computed, onMounted, ref } from 'vue'

use([
  CanvasRenderer,
  LineChart,
  PieChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  ToolboxComponent
])

const stats = ref({})
const chartData = ref([])
const modelData = ref([])
const channelData = ref([])
const revenueData = ref([])
const userGrowthData = ref([])
const tokenConsumptionData = ref([])
const recentRequests = ref([])
const timeRange = ref('today')
const revenueTimeRange = ref('7days')

const requestChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: chartData.value.map(item => item.time)
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '请求数',
        type: 'line',
        stack: 'Total',
        areaStyle: {
          opacity: 0.3
        },
        emphasis: {
          focus: 'series'
        },
        data: chartData.value.map(item => item.value),
        lineStyle: {
          color: '#409EFF'
        },
        itemStyle: {
          color: '#409EFF'
        }
      }
    ]
  }
})

const modelChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '模型使用',
        type: 'pie',
        radius: '50%',
        data: modelData.value.map(item => ({ name: item.model, value: item.count })),
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
})

const channelChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value'
    },
    yAxis: {
      type: 'category',
      data: channelData.value.map(item => item.name)
    },
    series: [
      {
        name: '请求数',
        type: 'bar',
        data: channelData.value.map(item => item.count),
        itemStyle: {
          color: function(params) {
            const colors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399']
            return colors[params.dataIndex % colors.length]
          }
        }
      }
    ]
  }
})

const revenueChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: revenueData.value.map(item => item.date)
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '收入',
        type: 'bar',
        data: revenueData.value.map(item => item.amount),
        itemStyle: {
          color: '#67C23A'
        }
      }
    ]
  }
})

const userGrowthChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: userGrowthData.value.map(item => item.date)
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '新增用户',
        type: 'line',
        data: userGrowthData.value.map(item => item.count),
        lineStyle: {
          color: '#409EFF'
        },
        itemStyle: {
          color: '#409EFF'
        }
      }
    ]
  }
})

const tokenConsumptionChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: tokenConsumptionData.value.map(item => item.date)
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: 'Token 消耗',
        type: 'line',
        data: tokenConsumptionData.value.map(item => item.count),
        lineStyle: {
          color: '#E6A23C'
        },
        itemStyle: {
          color: '#E6A23C'
        }
      }
    ]
  }
})

const loadStats = async () => {
  try {
    const response = await api.getDashboard()
    // 新响应格式: { code, message, data: {...} }
    stats.value = response.data?.data || response.data || {
      user_count: 120,
      user_growth: 5,
      channel_count: 20,
      channel_growth: 2,
      today_requests: 1500,
      request_growth: 150,
      total_tokens: 1200000,
      token_growth: 120000,
      total_revenue: 12000,
      revenue_growth: 1200,
      success_rate: 98.5,
      success_rate_change: 0.5,
      avg_response_time: 150,
      avg_response_time_change: -10
    }
  } catch (error) {
    console.error('加载统计数据失败:', error)
    // 模拟数据
    stats.value = {
      user_count: 120,
      user_growth: 5,
      channel_count: 20,
      channel_growth: 2,
      today_requests: 1500,
      request_growth: 150,
      total_tokens: 1200000,
      token_growth: 120000,
      total_revenue: 12000,
      revenue_growth: 1200,
      success_rate: 98.5,
      success_rate_change: 0.5,
      avg_response_time: 150,
      avg_response_time_change: -10
    }
  }
}

const loadChartData = async () => {
  try {
    // 模拟数据，实际应从API获取
    if (timeRange.value === 'today') {
      chartData.value = [
        { time: '00:00', value: 120 },
        { time: '04:00', value: 80 },
        { time: '08:00', value: 200 },
        { time: '12:00', value: 350 },
        { time: '16:00', value: 420 },
        { time: '20:00', value: 280 }
      ]
    } else if (timeRange.value === '7days') {
      chartData.value = [
        { time: 'Day 1', value: 1200 },
        { time: 'Day 2', value: 1350 },
        { time: 'Day 3', value: 1420 },
        { time: 'Day 4', value: 1500 },
        { time: 'Day 5', value: 1650 },
        { time: 'Day 6', value: 1480 },
        { time: 'Day 7', value: 1720 }
      ]
    } else if (timeRange.value === '30days') {
      chartData.value = [
        { time: 'Week 1', value: 8000 },
        { time: 'Week 2', value: 8500 },
        { time: 'Week 3', value: 9200 },
        { time: 'Week 4', value: 10000 }
      ]
    } else {
      chartData.value = [
        { time: 'Month 1', value: 30000 },
        { time: 'Month 2', value: 32000 },
        { time: 'Month 3', value: 35000 }
      ]
    }
  } catch (error) {
    console.error('加载图表数据失败:', error)
  }
}

const loadModelData = async () => {
  try {
    // 模拟数据，实际应从API获取
    modelData.value = [
      { model: 'gpt-3.5-turbo', count: 1200 },
      { model: 'gpt-4', count: 800 },
      { model: 'claude-3-sonnet', count: 600 },
      { model: 'gemini-pro', count: 400 },
      { model: 'gpt-4o', count: 300 }
    ]
  } catch (error) {
    console.error('加载模型数据失败:', error)
  }
}

const loadChannelData = async () => {
  try {
    // 模拟数据，实际应从API获取
    channelData.value = [
      { name: 'OpenAI', count: 1500 },
      { name: 'Claude', count: 800 },
      { name: 'Gemini', count: 600 },
      { name: 'Azure', count: 400 },
      { name: 'DeepSeek', count: 200 }
    ]
  } catch (error) {
    console.error('加载渠道数据失败:', error)
  }
}

const loadRevenueData = async () => {
  try {
    // 模拟数据，实际应从API获取
    if (revenueTimeRange.value === '7days') {
      revenueData.value = [
        { date: 'Day 1', amount: 1200 },
        { date: 'Day 2', amount: 1350 },
        { date: 'Day 3', amount: 1420 },
        { date: 'Day 4', amount: 1500 },
        { date: 'Day 5', amount: 1650 },
        { date: 'Day 6', amount: 1480 },
        { date: 'Day 7', amount: 1720 }
      ]
    } else if (revenueTimeRange.value === '30days') {
      revenueData.value = [
        { date: 'Week 1', amount: 8000 },
        { date: 'Week 2', amount: 8500 },
        { date: 'Week 3', amount: 9200 },
        { date: 'Week 4', amount: 10000 }
      ]
    } else {
      revenueData.value = [
        { date: 'Month 1', amount: 30000 },
        { date: 'Month 2', amount: 32000 },
        { date: 'Month 3', amount: 35000 }
      ]
    }
  } catch (error) {
    console.error('加载收入数据失败:', error)
  }
}

const loadUserGrowthData = async () => {
  try {
    // 模拟数据，实际应从API获取
    userGrowthData.value = [
      { date: 'Day 1', count: 10 },
      { date: 'Day 2', count: 15 },
      { date: 'Day 3', count: 8 },
      { date: 'Day 4', count: 20 },
      { date: 'Day 5', count: 12 },
      { date: 'Day 6', count: 18 },
      { date: 'Day 7', count: 25 }
    ]
  } catch (error) {
    console.error('加载用户增长数据失败:', error)
  }
}

const loadTokenConsumptionData = async () => {
  try {
    // 模拟数据，实际应从API获取
    tokenConsumptionData.value = [
      { date: 'Day 1', count: 120000 },
      { date: 'Day 2', count: 135000 },
      { date: 'Day 3', count: 142000 },
      { date: 'Day 4', count: 150000 },
      { date: 'Day 5', count: 165000 },
      { date: 'Day 6', count: 148000 },
      { date: 'Day 7', count: 172000 }
    ]
  } catch (error) {
    console.error('加载Token消耗数据失败:', error)
  }
}

const loadRecentRequests = async () => {
  try {
    // 模拟数据，实际应从API获取
    recentRequests.value = [
      { id: 1, user_email: 'user1@example.com', model_name: 'gpt-3.5-turbo', path: '/chat/completions', status_code: 200, duration: 120, created_at: new Date().toISOString() },
      { id: 2, user_email: 'user2@example.com', model_name: 'gpt-4', path: '/chat/completions', status_code: 200, duration: 250, created_at: new Date().toISOString() },
      { id: 3, user_email: 'user3@example.com', model_name: 'claude-3-sonnet', path: '/chat/completions', status_code: 200, duration: 180, created_at: new Date().toISOString() },
      { id: 4, user_email: 'user4@example.com', model_name: 'gemini-pro', path: '/chat/completions', status_code: 200, duration: 150, created_at: new Date().toISOString() },
      { id: 5, user_email: 'user5@example.com', model_name: 'gpt-3.5-turbo', path: '/chat/completions', status_code: 401, duration: 50, created_at: new Date().toISOString() },
      { id: 6, user_email: 'user6@example.com', model_name: 'gpt-4o', path: '/chat/completions', status_code: 200, duration: 200, created_at: new Date().toISOString() }
    ]
  } catch (error) {
    console.error('加载最近请求失败:', error)
  }
}

const refreshRequests = async () => {
  await loadRecentRequests()
}

const refreshChannelData = async () => {
  await loadChannelData()
}

onMounted(async () => {
  await Promise.all([
    loadStats(),
    loadChartData(),
    loadModelData(),
    loadChannelData(),
    loadRevenueData(),
    loadUserGrowthData(),
    loadTokenConsumptionData(),
    loadRecentRequests()
  ])
})
</script>

<style scoped lang="scss">
.admin-dashboard {
  padding: 0;
  
  .stat-card {
    transition: all 0.3s ease;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }
    
    .stat-content {
      display: flex;
      align-items: center;
      gap: 16px;
      
      .stat-icon {
        width: 60px;
        height: 60px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        
        .el-icon {
          font-size: 28px;
          color: #fff;
        }
      }
      
      .stat-info {
        flex: 1;
        
        .stat-value {
          font-size: 24px;
          font-weight: 600;
          color: #303133;
        }
        
        .stat-label {
          font-size: 14px;
          color: #909399;
          margin-top: 4px;
        }
        
        .stat-change {
          font-size: 12px;
          margin-top: 2px;
          
          &.positive {
            color: #67C23A;
          }
          
          &.negative {
            color: #F56C6C;
          }
        }
      }
    }
  }
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    
    span {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }
  }
  
  .chart-container {
    padding: 20px;
    height: 300px;
  }
  
  .el-table {
    margin-top: 10px;
    
    .el-table__row {
      transition: all 0.3s ease;
      
      &:hover {
        background-color: #f5f7fa;
      }
    }
  }
}
</style>
