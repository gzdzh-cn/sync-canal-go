import{ag as G,a1 as V,X,bd as q,be as J,bf as Q,r as B,u as k,m as U,a as W,j as u,t as g,G as _,o as Y,aB as ee,R as d,k as j,e as M,i as re}from"./index.e73d7bc6.js";var ae=Object.prototype,te=ae.hasOwnProperty,ne=G(function(b,s){if(V(s)||X(s)){q(s,J(s),b);return}for(var e in s)te.call(s,e)&&Q(b,e,s[e])}),T=ne;/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var se={bordered:!1,column:2,itemLayout:"horizontal",layout:"horizontal",size:"medium",tableLayout:"auto"},$={span:1};/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var H=B.exports.createContext(null);/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var S=function(){return null};S.displayName="DescriptionsItem";/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var K=function(s){var e=s.row,w=k(),E=w.classPrefix,N=U("descriptions"),x=W(N,2),C=x[0],z=x[1],l=B.exports.useContext(H),P="".concat(E,"-descriptions"),m=function(r){var a=arguments.length>1&&arguments[1]!==void 0?arguments[1]:"horizontal",h=arguments.length>2?arguments[2]:void 0,t=r.span,c=a==="horizontal"?1:t;return g("td",{colSpan:c,className:"".concat(P,"__label"),style:l.labelStyle,children:[r.label,l.colon&&z(C.colonText)]},h)},f=function(r){var a=arguments.length>1&&arguments[1]!==void 0?arguments[1]:"horizontal",h=arguments.length>2?arguments[2]:void 0,t=r.span,c=t>1&&a==="horizontal"?t*2-1:t;return u("td",{colSpan:c,className:"".concat(P,"__content"),style:l.contentStyle,children:r.content},h)},R=function(){return u("tr",{children:e.map(function(r,a){return g(_,{children:[m(r),f(r)]})})})},v=function(){return g(_,{children:[u("tr",{children:e.map(function(r,a){return m(r,"vertical","top_".concat(a))})}),u("tr",{children:e.map(function(r,a){return f(r,"vertical","bottom_".concat(a))})})]})},L=function(){return u(_,{children:e.map(function(r,a){return g("tr",{children:[m(r),f(r)]},a)})})},I=function(){return u(_,{children:e.map(function(r,a){return g(_,{children:[u("tr",{children:m(r)}),u("tr",{children:f(r)})]})})})};return l.layout==="horizontal"?l.itemLayout==="horizontal"?R():v():l.itemLayout==="horizontal"?L():I()};K.displayName="DescriptionsRow";/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var A=function(s){var e=Y(s,se),w=e.className,E=e.style,N=e.title,x=e.bordered,C=e.column,z=e.layout,l=e.items,P=e.children,m=e.tableLayout,f=k(),R=f.classPrefix,v="".concat(R,"-descriptions"),L=ee(),I=L.SIZE,p=function(){var t=[];if(re(l))t=l.map(function(n){var o=T({},$,n),i=o.span;return{label:n.label,content:n.content,span:i}});else{var c=d.Children.toArray(P).filter(function(n){var o;return((o=n.type)===null||o===void 0?void 0:o.displayName)===S.displayName});c.length!==0&&(t=c.map(function(n){var o,i=n.props,Z=T({},$,i),F=Z.span;return{label:i.label,content:(o=i.content)!==null&&o!==void 0?o:i.children,span:F}}))}if(z==="vertical")return[t];var y=[],D=C,O=[];return t.forEach(function(n,o){var i=n.span;D>=i?(y.push(n),D-=i):(O.push(y),y=[n],D=C-i),o===t.length-1&&(Reflect.set(n,"span",i+D),O.push(y))}),O},r=function(){return N?d.createElement("div",{className:"".concat(v,"__header")},N):""},a=function(){var t=["".concat(v,"__body"),I[e.size],M({},"".concat(v,"__body--fixed"),m==="fixed"),M({},"".concat(v,"__body--border"),x)];return d.createElement("table",{className:j(t)},d.createElement("tbody",null,p().map(function(c,y){return d.createElement(K,{row:c,key:y})})))};return d.createElement(H.Provider,{value:e},d.createElement("div",{className:j(w,v),style:E},r(),a()))};A.displayName="Descriptions";A.DescriptionsItem=S;/**
 * tdesign v1.16.8
 * (c) 2026 tdesign
 * @license MIT
 */var ie=A;export{ie as D};
