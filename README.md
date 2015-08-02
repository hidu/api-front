# api-manager

<h1 class="page-header">概述</h1>
<p>api manager 是用来做api请求代理转发的，<code>每一个请求会同时发送到所有的后端</code>，然后选取其中一个后端的相应返回给client。</p>

<p>也就是，每一个后端服务都可以接收到全量的数据。</p>

<h1 class="page-header">用途</h1>
<h3>多人多模块开发演示</h3>
<p><code>1.一种情况：2个模块分别2个人开发，2个环境都能收到数据。</code></p>
<p><img src="/res/img/useage_0.png"></p>
<p><code>2.另一种情况：合作有交叉，下游一个人就收不到数据了。</code></p>
<p><img src="/res/img/useage_1.png"></p>
<p><code>3.一个理想的情况：合作有交叉时，下游每个人都能收到数据。</code></p>
<p><img src="/res/img/useage_2.png"></p>
