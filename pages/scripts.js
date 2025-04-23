// 页面加载完成后初始化所有功能
document.addEventListener('DOMContentLoaded', () => {
    // 初始化所有功能
    [
        initParticles,
        initScrollAnimations,
        initCardHover,
        initNavbar,
        initSmoothScroll,
        initTerminalTyping,
        initBackToTop,
        initMobileMenu,
        initLanguageToggle,
        initArchitectureDiagram
    ].forEach(fn => fn());
});

// 初始化粒子背景
function initParticles() {
    particlesJS('particles-js', {
        particles: {
            number: { value: 80, density: { enable: true, value_area: 800 } },
            color: { value: "#6875F5" },
            shape: {
                type: "circle",
                stroke: { width: 0, color: "#000000" },
                polygon: { nb_sides: 5 }
            },
            opacity: {
                value: 0.5,
                random: true,
                anim: { enable: true, speed: 1, opacity_min: 0.1, sync: false }
            },
            size: {
                value: 3,
                random: true,
                anim: { enable: false, speed: 40, size_min: 0.1, sync: false }
            },
            line_linked: {
                enable: true,
                distance: 150,
                color: "#6875F5",
                opacity: 0.4,
                width: 1
            },
            move: {
                enable: true,
                speed: 2,
                direction: "none",
                random: false,
                straight: false,
                out_mode: "out",
                bounce: false,
                attract: { enable: false, rotateX: 600, rotateY: 1200 }
            }
        },
        interactivity: {
            detect_on: "canvas",
            events: {
                onhover: { enable: false, mode: "grab" },
                onclick: { enable: false, mode: "push" },
                resize: true
            },
            modes: {
                grab: { distance: 140, line_linked: { opacity: 1 } },
                bubble: { distance: 400, size: 40, duration: 2, opacity: 8, speed: 3 },
                repulse: { distance: 200, duration: 0.4 },
                push: { particles_nb: 4 },
                remove: { particles_nb: 2 }
            }
        },
        retina_detect: true
    });
}

// 初始化滚动动画
function initScrollAnimations() {
    gsap.registerPlugin(ScrollTrigger);
    
    // Hero 部分动画
    const heroElements = [
        {el: '.logo-container', delay: 0.3},
        {el: '.hero-title', delay: 0.6},
        {el: '.hero-description', delay: 0.9},
        {el: '.hero-buttons', delay: 1.2}
    ];
    
    heroElements.forEach(item => {
        gsap.to(item.el, {
            opacity: 1,
            y: 0,
            duration: 1,
            delay: item.delay
        });
    });
    
    // 通用滚动动画函数
    const createScrollAnimation = (selector, stagger = 0) => {
        const elements = document.querySelectorAll(selector);
        elements.forEach((el, index) => {
            const delay = el.dataset.delay ? parseFloat(el.dataset.delay) : index * stagger;
            
            gsap.fromTo(el, 
                { opacity: 0, y: 30 },
                { 
                    opacity: 1, 
                    y: 0, 
                    duration: 0.8,
                    delay: delay,
                    scrollTrigger: {
                        trigger: el,
                        start: "top 85%",
                        toggleActions: "play none none none"
                    }
                }
            );
        });
    };
    
    // 应用滚动动画到各部分
    createScrollAnimation('.feature-card', 0.2);
    createScrollAnimation('.demo-title');
    createScrollAnimation('.demo-content');
    createScrollAnimation('.resources-title');
    createScrollAnimation('.resources-links');
    createScrollAnimation('.download-section');
}

// 特性卡片鼠标悬停3D效果
function initCardHover() {
    document.querySelectorAll('.card-3d').forEach(card => {
        const handleMouseMove = (e) => {
            const rect = card.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
        
            const midCardWidth = rect.width / 2;
            const midCardHeight = rect.height / 2;
            
            const angleY = -(((x - midCardWidth) / midCardWidth) * 10);
            const angleX = ((y - midCardHeight) / midCardHeight) * 10;
            
            card.style.transform = `perspective(1000px) rotateX(${angleX}deg) rotateY(${angleY}deg)`;
        };
        
        card.addEventListener('mousemove', handleMouseMove);
        card.addEventListener('mouseleave', () => {
            card.style.transform = 'perspective(1000px) rotateX(0) rotateY(0)';
        });
    });
}

// 导航栏滚动效果和平滑滚动
function initNavbar() {
    const navbar = document.getElementById('navbar');
    
    window.addEventListener('scroll', () => {
        navbar.classList.toggle('scrolled', window.scrollY > 50);
    });
}

// 平滑滚动
function initSmoothScroll() {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            
            // 关闭移动端菜单（如果打开）
            const mobileMenu = document.getElementById('mobile-menu');
            if (mobileMenu.classList.contains('block')) {
                mobileMenu.classList.replace('block', 'hidden');
            }
            
            const targetId = this.getAttribute('href');
            const targetElement = document.querySelector(targetId);
            
            if (targetElement) {
                gsap.to(window, {
                    duration: 1,
                    scrollTo: {
                        y: targetElement,
                        offsetY: 80
                    },
                    ease: "power2.inOut"
                });
            }
        });
    });
}

// 终端打字效果
function initTerminalTyping() {
    const terminalText = document.getElementById('terminal-text');
    if (!terminalText) return;
    
    const text = terminalText.textContent;
    terminalText.textContent = '';
    
    let i = 0;
    let isTagOpen = false;
    
    function typeWriter() {
        if (i < text.length) {
            const char = text.charAt(i);
            
            // 处理HTML标签
            if (char === '<') isTagOpen = true;
            if (char === '>') isTagOpen = false;
            
            // 如果当前字符属于HTML标签，不做延迟
            if (isTagOpen) {
                while (i < text.length && text.charAt(i) !== '>') {
                    terminalText.innerHTML += text.charAt(i);
                    i++;
                }
                if (i < text.length) {
                    terminalText.innerHTML += text.charAt(i); // 添加 '>'
                    i++;
                }
                requestAnimationFrame(typeWriter);
            } else {
                // 普通文本字符一个个打出
                terminalText.innerHTML += char;
                i++;
                
                // 动态调整速度
                let speed = 20;
                if (char === '#') speed = 50 + Math.random() * 100;
                else if (char === '$') speed = 100 + Math.random() * 200;
                else if (char === '\n') speed = 300 + Math.random() * 200;
                else speed = 10 + Math.random() * 30;
                
                setTimeout(typeWriter, speed);
            }
        }
    }
    
    // 交叉观察器优化
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                typeWriter();
                observer.unobserve(entry.target);
            }
        });
    }, { threshold: 0.5 });
    
    observer.observe(terminalText.parentElement.parentElement);
}

// 返回顶部按钮
function initBackToTop() {
    const backToTopButton = document.getElementById('back-to-top');
    if (!backToTopButton) return;
    
    const toggleButtonVisibility = () => {
        const isVisible = window.scrollY > 300;
        if (isVisible) {
            backToTopButton.classList.remove('hidden');
            setTimeout(() => backToTopButton.classList.remove('opacity-0'), 10);
        } else {
            backToTopButton.classList.add('opacity-0');
            setTimeout(() => backToTopButton.classList.add('hidden'), 300);
        }
    };
    
    window.addEventListener('scroll', toggleButtonVisibility);
    
    backToTopButton.addEventListener('click', () => {
        gsap.to(window, {
            duration: 1.5, 
            scrollTo: 0,
            ease: "power2.out"
        });
    });
}

// 移动菜单交互
function initMobileMenu() {
    const mobileMenuButton = document.getElementById('mobile-menu-button');
    const mobileMenu = document.getElementById('mobile-menu');
    
    if (mobileMenuButton && mobileMenu) {
        mobileMenuButton.addEventListener('click', () => {
            mobileMenu.classList.toggle('hidden');
        });
    }
}

// 语言切换功能
function initLanguageToggle() {
    // 翻译数据
    const translations = {
        'en': {
            'features': 'Features',
            'architecture': 'Architecture',
            'documentation': 'Documentation',
            'repository': 'Repository',
            'title': 'NodePass',
            'subtitle': 'NodePass is a secure, efficient TCP/UDP tunneling solution that delivers fast, reliable access across network restrictions using pre-established TLS/TCP connections.',
            'github': 'GitHub Project',
            'learn-more': 'Learn More',
            'features-title': 'Core Features',
            'feature1-title': 'High-Performance Tunnel',
            'feature1-desc': 'Lightweight tunnel implemented in Go, with extremely low latency and high throughput, supporting large-scale concurrent connections.',
            'feature2-title': 'Three-Layer Architecture',
            'feature2-desc': 'Innovative master, server, and client three-layer architecture, achieving flexible network topology and unified management.',
            'feature3-title': 'Multi-Protocol Support',
            'feature3-desc': 'Supports TCP and UDP protocols, meeting network requirements for different application scenarios, with flexible configuration options.',
            'feature4-title': 'Secure and Reliable',
            'feature4-desc': 'Built-in TLS encryption and authentication mechanisms ensure the security and integrity of data transmission, preventing unauthorized access.',
            'feature5-title': 'Easy Configuration',
            'feature5-desc': 'Simple command-line interface without configuration files, making deployment and management simple and efficient, suitable for users of all technical levels.',
            'feature6-title': 'Cross-Platform Compatible',
            'feature6-desc': 'Supports Linux, Windows, macOS, and other operating systems, can be seamlessly deployed and run in various environments.',
            'how-it-works': 'NodePass Architecture',
            'node-architecture': 'Node Architecture',
            'architecture-desc': 'NodePass uses a unique three-layer architecture:',
            'master-node-1': 'Master API',
            'master-node-2': 'Master API',
            'server-component': 'Server',
            'server-desc': 'Handles connections and protocol translation',
            'client-component': 'Client',
            'client-desc': 'Establishes and maintains tunnel connections',
            'connection-channels': 'Connection Channels',
            'data-flow': 'Data Transmission Flow',
            'data-flow-desc': 'NodePass establishes peer-to-peer bidirectional data flow between instances:',
            'control-channel': 'Control Channel',
            'control-channel-1': 'Unencrypted TCP connection for signaling',
            'control-channel-2': 'Persistent connection for tunnel lifetime',
            'control-channel-3': 'URL-based signaling protocol',
            'control-channel-4': 'Coordinates connection tunnel establishment',
            'data-channel': 'Data Channel',
            'data-channel-1': 'Configurable TLS encryption (3 modes)',
            'data-channel-2': 'Created on-demand for each connection',
            'data-channel-3': 'Efficient connection pooling system',
            'data-channel-4': 'Supports both TCP and UDP protocols',
            'security-features': 'Security Features',
            'security-mode-0': 'Mode 0',
            'security-mode-0-desc': 'Unencrypted data transfer (fastest, least secure)',
            'security-mode-1': 'Mode 1',
            'security-mode-1-desc': 'Self-signed certificate encryption (good security, no verification)',
            'security-mode-2': 'Mode 2',
            'security-mode-2-desc': 'Verified certificate encryption (highest security, requires valid certificates)',
            'architecture-benefit': 'This peer-to-peer architecture allows NodePass to flexibly adapt to various network environments, achieving efficient data transmission between endpoints with enhanced security and protocol translation.',
            'resources-title': 'Resources & Documentation',
            'doc1-title': 'Installation Guide',
            'doc1-desc': 'Detailed installation steps and system requirements to help you quickly deploy NodePass.',
            'doc2-title': 'API Documentation',
            'doc2-desc': 'Detailed description of Restful API for building custom applications and integrations.',
            'doc3-title': 'Configuration Guide',
            'doc3-desc': 'Comprehensive explanation of configuration options to help you customize NodePass according to your needs.',
            'doc4-title': 'Usage Examples',
            'doc4-desc': 'Common application scenario examples to help you quickly get started and apply to actual projects.',
            'doc5-title': 'Troubleshooting',
            'doc5-desc': 'FAQs and troubleshooting guide to solve problems you may encounter when using NodePass.',
            'doc6-title': 'GitHub Repository',
            'doc6-desc': 'Visit the GitHub repository to get the latest code, report issues, or contribute code.',
            'start-using': 'Start Using NodePass',
            'download-desc': 'Download and deploy NodePass now to experience an efficient and secure network tunnel solution.',
            'download-latest': 'Download Latest Version',
            'view-installation': 'View Installation Guide',
            'footer-desc': 'Universal TCP/UDP Tunneling Solution'
        },
        'zh': {
            'features': '特性',
            'architecture': '架构',
            'documentation': '文档',
            'repository': '仓库',
            'title': 'NodePass',
            'subtitle': '通用TCP/UDP隧道解决方案，免配置单文件多模式，采用控制数据双路分离架构，内置零延迟自适应连接池，实现跨网络限制的快速安全访问。',
            'github': 'GitHub项目',
            'learn-more': '了解更多',
            'features-title': '核心特性',
            'feature1-title': '高性能隧道',
            'feature1-desc': '使用Go语言实现的轻量级隧道，具有极低的延迟和高吞吐量，支持大规模并发连接。',
            'feature2-title': '三层架构',
            'feature2-desc': '创新的主控、服务端、客户端三层架构，实现灵活的网络拓扑和统一管理。',
            'feature3-title': '多协议支持',
            'feature3-desc': '支持TCP和UDP协议，满足不同应用场景的网络需求，具有灵活的配置选项。',
            'feature4-title': '安全可靠',
            'feature4-desc': '内置TLS加密和认证机制，确保数据传输的安全性和完整性，防止未授权访问。',
            'feature5-title': '易于配置',
            'feature5-desc': '简单的命令行界面无需配置文件，使部署和管理变得简单高效，适合各技术水平用户。',
            'feature6-title': '跨平台兼容',
            'feature6-desc': '支持Linux、Windows、macOS等操作系统，可以在各种环境中无缝部署和运行。',
            'how-it-works': 'NodePass架构',
            'node-architecture': '节点架构',
            'architecture-desc': 'NodePass使用独特的三层架构：',
            'master-node-1': '主控API',
            'master-node-2': '主控API',
            'server-component': '服务端',
            'server-desc': '处理传入连接和协议转换',
            'client-component': '客户端',
            'client-desc': '建立和维护隧道连接',
            'connection-channels': '连接通道',
            'data-flow': '数据传输流程',
            'data-flow-desc': 'NodePass在实例之间建立点对点双向数据流：',
            'control-channel': '控制通道',
            'control-channel-1': '用于信令的非加密TCP连接',
            'control-channel-2': '隧道生命周期内的持久连接',
            'control-channel-3': '基于URL的信令协议',
            'control-channel-4': '协调连接隧道建立',
            'data-channel': '数据通道',
            'data-channel-1': '可配置的TLS加密（3种模式）',
            'data-channel-2': '按需为每个连接创建',
            'data-channel-3': '高效的连接池系统',
            'data-channel-4': '同时支持TCP和UDP协议',
            'security-features': '安全特性',
            'security-mode-0': '模式 0',
            'security-mode-0-desc': '非加密数据传输（最快，安全性最低）',
            'security-mode-1': '模式 1',
            'security-mode-1-desc': '自签名证书加密（良好安全性，无验证）',
            'security-mode-2': '模式 2',
            'security-mode-2-desc': '已验证证书加密（最高安全性，需要有效证书）',
            'architecture-benefit': '这种点对点架构使NodePass能够灵活适应各种网络环境，通过增强的安全性和协议转换实现端点之间的高效数据传输。',
            'resources-title': '资源与文档',
            'doc1-title': '安装指南',
            'doc1-desc': '详细的安装步骤和系统要求，帮助您快速部署NodePass。',
            'doc2-title': 'API文档',
            'doc2-desc': '详细描述Restful API，用于构建自定义应用程序和集成。',
            'doc3-title': '配置指南',
            'doc3-desc': '全面解释配置选项，帮助您按需自定义NodePass行为。',
            'doc4-title': '使用示例',
            'doc4-desc': '常见应用场景用例，帮助您快速上手并应用到实际项目中。',
            'doc5-title': '故障排除',
            'doc5-desc': '常见问题和排查指南，解决NodePass使用时遇到的问题。',
            'doc6-title': 'GitHub仓库',
            'doc6-desc': '访问GitHub仓库获取最新代码、报告问题或贡献代码。',
            'start-using': '开始使用NodePass',
            'download-desc': '立即下载并部署NodePass，体验高效安全的网络隧道解决方案。',
            'download-latest': '下载最新版本',
            'view-installation': '查看安装指南',
            'footer-desc': '通用TCP/UDP隧道解决方案'
        }
    };
    
    // 语言切换函数
    const toggleAndApplyLanguage = (currentLang) => {
        const newLang = currentLang === 'en' ? 'zh' : 'en';
        applyTranslation(newLang);
        return newLang;
    };
    
    // 应用翻译
    const applyTranslation = (lang) => {
        // 更新文本内容
        document.querySelectorAll('[data-i18n]').forEach(element => {
            const key = element.getAttribute('data-i18n');
            if (translations[lang][key]) {
                element.textContent = translations[lang][key];
            }
        });
        
        // 更新语言显示
        ['current-lang', 'mobile-current-lang'].forEach(id => {
            const langElem = document.getElementById(id);
            if (langElem) langElem.textContent = lang.toUpperCase();
        });
        
        // 更新文档链接
        document.querySelectorAll('[data-langurl-en]').forEach(link => {
            const enUrl = link.getAttribute('data-langurl-en');
            const zhUrl = link.getAttribute('data-langurl-zh');
            link.setAttribute('href', lang === 'en' ? enUrl : zhUrl);
        });
        
        // 保存语言设置
        localStorage.setItem('nodepass-lang', lang);
        
        return lang;
    };
    
    // 获取初始语言
    let currentLang = localStorage.getItem('nodepass-lang') || 'en';
    if (!['en', 'zh'].includes(currentLang)) currentLang = 'en';
    
    // 应用初始语言
    applyTranslation(currentLang);
    
    // 添加事件监听器
    ['language-toggle', 'mobile-language-toggle'].forEach(id => {
        const toggle = document.getElementById(id);
        if (toggle) {
            toggle.addEventListener('click', () => {
                currentLang = toggleAndApplyLanguage(currentLang);
            });
        }
    });
}

// 初始化架构图动画
function initArchitectureDiagram() {
    // 检查架构图是否存在
    const diagram = document.querySelector('.architecture-diagram');
    if (!diagram) return;
    
    // 绘制连接线
    drawConnectionLines();
    
    // 响应窗口大小变化
    window.addEventListener('resize', debounce(drawConnectionLines, 250));
}

// 简单的防抖函数
function debounce(func, wait) {
    let timeout;
    return function() {
        const context = this;
        const args = arguments;
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(context, args), wait);
    };
}

// 绘制节点之间的连接线
function drawConnectionLines() {
    const connectionLinesContainer = document.getElementById('connection-lines');
    if (!connectionLinesContainer) return;
    
    // 清空现有连接线
    connectionLinesContainer.innerHTML = '';
    
    // 获取节点位置
    const masterTopNode = document.querySelector('.arch-layer[data-node-type="master-top"]');
    const masterBottomNode = document.querySelector('.arch-layer[data-node-type="master-bottom"]');
    
    if (!masterTopNode || !masterBottomNode) return;
    
    const createSvg = () => {
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('width', '100%');
        svg.setAttribute('height', '100%');
        svg.style.position = 'absolute';
        svg.style.top = '0';
        svg.style.left = '0';
        svg.style.pointerEvents = 'none';
        return svg;
    };
    
    const getElementRect = (element, containerRect) => {
        const rect = element.getBoundingClientRect();
        return {
            left: rect.left - containerRect.left,
            right: rect.right - containerRect.left,
            top: rect.top - containerRect.top,
            bottom: rect.bottom - containerRect.top,
            width: rect.width,
            height: rect.height
        };
    };
    
    const createPath = (startX, startY, endX, endY, offset, color, isDashed) => {
        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        const midY = (startY + endY) / 2;
        
        path.setAttribute('d', `M${startX + offset},${startY} 
                            C${startX + offset},${midY} 
                            ${endX + offset},${midY} 
                            ${endX + offset},${endY}`);
        path.setAttribute('stroke', color);
        path.setAttribute('stroke-width', '2');
        path.setAttribute('fill', 'none');
        
        if (isDashed) {
            path.setAttribute('stroke-dasharray', '4,4');
            
            // 添加动画
            const animate = document.createElementNS('http://www.w3.org/2000/svg', 'animate');
            animate.setAttribute('attributeName', 'stroke-dashoffset');
            animate.setAttribute('from', '0');
            animate.setAttribute('to', '16');
            animate.setAttribute('dur', '1s');
            animate.setAttribute('repeatCount', 'indefinite');
            path.appendChild(animate);
        }
        
        return path;
    };
    
    // 绘制连接线
    const svg = createSvg();
    const containerRect = connectionLinesContainer.getBoundingClientRect();
    
    // 获取各元素位置
    const topServerElement = masterTopNode.querySelector('.bg-indigo-900');
    const topClientElement = masterTopNode.querySelector('.bg-green-900');
    const bottomServerElement = masterBottomNode.querySelector('.bg-indigo-900');
    const bottomClientElement = masterBottomNode.querySelector('.bg-green-900');
    
    if (!topServerElement || !topClientElement || !bottomServerElement || !bottomClientElement) return;
    
    const topServerRect = getElementRect(topServerElement, containerRect);
    const topClientRect = getElementRect(topClientElement, containerRect);
    const bottomServerRect = getElementRect(bottomServerElement, containerRect);
    const bottomClientRect = getElementRect(bottomClientElement, containerRect);
    
    // 计算连接点
    const topServerBottom = {
        x: topServerRect.left + topServerRect.width / 2,
        y: topServerRect.bottom
    };
    
    const topClientBottom = {
        x: topClientRect.left + topClientRect.width / 2,
        y: topClientRect.bottom
    };
    
    const bottomServerTop = {
        x: bottomServerRect.left + bottomServerRect.width / 2,
        y: bottomServerRect.top
    };
    
    const bottomClientTop = {
        x: bottomClientRect.left + bottomClientRect.width / 2,
        y: bottomClientRect.top
    };
    
    // 创建连接线路径
    const offset = 15;
    
    // Server > Client 连接
    svg.appendChild(createPath(
        topServerBottom.x, topServerBottom.y,
        bottomClientTop.x, bottomClientTop.y,
        -offset, '#3B82F6', true // 蓝色控制通道
    ));
    
    svg.appendChild(createPath(
        topServerBottom.x, topServerBottom.y,
        bottomClientTop.x, bottomClientTop.y,
        offset, '#10B981', true // 绿色数据通道
    ));
    
    // Client > Server 连接
    svg.appendChild(createPath(
        topClientBottom.x, topClientBottom.y,
        bottomServerTop.x, bottomServerTop.y,
        -offset, '#3B82F6', true // 蓝色控制通道
    ));
    
    svg.appendChild(createPath(
        topClientBottom.x, topClientBottom.y,
        bottomServerTop.x, bottomServerTop.y,
        offset, '#10B981', true // 绿色数据通道
    ));
    
    // 添加到容器
    connectionLinesContainer.appendChild(svg);
}