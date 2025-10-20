#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Pixly UI自动化测试系统
使用真实的鼠标点击和键盘输入进行测试，避免交互问题
"""

import sys
import os
import time
import subprocess
import shutil
import json
import logging
from pathlib import Path
from typing import Dict, List, Optional, Tuple
import threading
from datetime import datetime

try:
    import pyautogui
    import psutil
    from PIL import Image, ImageDraw, ImageFont
except ImportError:
    print("请安装必需的依赖: pip install pyautogui psutil pillow")
    sys.exit(1)

# 配置pyautogui
pyautogui.FAILSAFE = True  # 鼠标移动到左上角时停止
pyautogui.PAUSE = 0.5      # 每次操作间隔0.5秒

class PixlyUIAutomation:
    """Pixly UI自动化测试类"""
    
    def __init__(self, pixly_binary_path: str, test_data_dir: str, results_dir: str):
        self.pixly_binary_path = Path(pixly_binary_path)
        self.test_data_dir = Path(test_data_dir)
        self.results_dir = Path(results_dir)
        self.process = None
        self.log_file = None
        
        # 创建结果目录
        self.results_dir.mkdir(parents=True, exist_ok=True)
        
        # 设置日志
        log_file = self.results_dir / f"automation_test_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(log_file, encoding='utf-8'),
                logging.StreamHandler(sys.stdout)
            ]
        )
        self.logger = logging.getLogger(__name__)
        
        # 测试状态
        self.test_results = {
            'start_time': datetime.now().isoformat(),
            'tests': [],
            'summary': {
                'total': 0,
                'passed': 0,
                'failed': 0,
                'errors': []
            }
        }
        
    def setup_test_environment(self) -> bool:
        """设置测试环境"""
        try:
            # 检查Pixly二进制文件
            if not self.pixly_binary_path.exists():
                self.logger.error(f"Pixly二进制文件不存在: {self.pixly_binary_path}")
                return False
                
            # 检查测试数据目录
            if not self.test_data_dir.exists():
                self.logger.error(f"测试数据目录不存在: {self.test_data_dir}")
                return False
                
            # 创建测试工作目录
            self.work_dir = self.results_dir / "test_workspace"
            if self.work_dir.exists():
                shutil.rmtree(self.work_dir)
            self.work_dir.mkdir(parents=True)
            
            self.logger.info("测试环境设置完成")
            return True
            
        except Exception as e:
            self.logger.error(f"设置测试环境失败: {e}")
            return False
    
    def prepare_test_data(self, test_name: str) -> Path:
        """为每个测试准备独立的数据副本"""
        test_data_copy = self.work_dir / f"{test_name}_data"
        if test_data_copy.exists():
            shutil.rmtree(test_data_copy)
        
        # 复制测试数据
        shutil.copytree(self.test_data_dir, test_data_copy)
        self.logger.info(f"为测试 {test_name} 准备数据: {test_data_copy}")
        return test_data_copy
    
    def scan_directory_files(self, directory: Path) -> Dict[str, List[str]]:
        """扫描目录中的文件并按格式分类"""
        files_by_format = {}
        
        for file_path in directory.rglob('*'):
            if file_path.is_file() and not file_path.name.startswith('.'):
                suffix = file_path.suffix.lower()
                if suffix not in files_by_format:
                    files_by_format[suffix] = []
                files_by_format[suffix].append(str(file_path))
        
        return files_by_format
    
    def start_pixly_process(self) -> bool:
        """启动Pixly进程"""
        try:
            # 构建启动命令
            cmd = [str(self.pixly_binary_path)]
            
            self.logger.info(f"启动Pixly进程: {' '.join(cmd)}")
            
            # 启动进程
            self.process = subprocess.Popen(
                cmd,
                cwd=self.pixly_binary_path.parent,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                bufsize=1,
                universal_newlines=True
            )
            
            # 等待进程启动
            time.sleep(3)
            
            if self.process.poll() is None:
                self.logger.info("Pixly进程启动成功")
                return True
            else:
                stdout, stderr = self.process.communicate()
                self.logger.error(f"Pixly进程启动失败:\nSTDOUT: {stdout}\nSTDERR: {stderr}")
                return False
                
        except Exception as e:
            self.logger.error(f"启动Pixly进程时出错: {e}")
            return False
    
    def send_input_to_process(self, input_text: str, delay: float = 1.0):
        """向进程发送输入"""
        try:
            if self.process and self.process.poll() is None:
                # 使用键盘输入模拟
                pyautogui.write(input_text, interval=0.1)
                pyautogui.press('enter')
                time.sleep(delay)
                self.logger.info(f"发送输入: {input_text}")
            else:
                self.logger.error("进程未运行，无法发送输入")
        except Exception as e:
            self.logger.error(f"发送输入失败: {e}")
    
    def simulate_user_interaction(self, test_config: Dict) -> bool:
        """模拟用户交互"""
        try:
            steps = test_config.get('steps', [])
            
            for i, step in enumerate(steps):
                step_type = step.get('type')
                step_data = step.get('data')
                delay = step.get('delay', 1.0)
                
                self.logger.info(f"执行步骤 {i+1}: {step_type}")
                
                if step_type == 'input':
                    self.send_input_to_process(step_data, delay)
                    
                elif step_type == 'select':
                    # 模拟选择操作
                    pyautogui.write(str(step_data), interval=0.1)
                    pyautogui.press('enter')
                    time.sleep(delay)
                    
                elif step_type == 'wait':
                    # 等待指定时间
                    time.sleep(step_data)
                    
                elif step_type == 'screenshot':
                    # 截图记录
                    screenshot = pyautogui.screenshot()
                    screenshot_path = self.results_dir / f"screenshot_step_{i+1}.png"
                    screenshot.save(screenshot_path)
                    self.logger.info(f"截图保存: {screenshot_path}")
                    
                else:
                    self.logger.warning(f"未知步骤类型: {step_type}")
            
            return True
            
        except Exception as e:
            self.logger.error(f"模拟用户交互失败: {e}")
            return False
    
    def monitor_process_output(self, timeout: int = 300) -> Tuple[str, str]:
        """监控进程输出"""
        start_time = time.time()
        stdout_lines = []
        stderr_lines = []
        
        try:
            while time.time() - start_time < timeout:
                if self.process.poll() is not None:
                    # 进程已结束
                    break
                    
                # 读取输出
                if self.process.stdout:
                    line = self.process.stdout.readline()
                    if line:
                        stdout_lines.append(line.strip())
                        self.logger.info(f"STDOUT: {line.strip()}")
                
                if self.process.stderr:
                    line = self.process.stderr.readline()
                    if line:
                        stderr_lines.append(line.strip())
                        self.logger.warning(f"STDERR: {line.strip()}")
                
                time.sleep(0.1)
            
            # 获取剩余输出
            if self.process:
                remaining_stdout, remaining_stderr = self.process.communicate(timeout=5)
                if remaining_stdout:
                    stdout_lines.extend(remaining_stdout.split('\n'))
                if remaining_stderr:
                    stderr_lines.extend(remaining_stderr.split('\n'))
            
            return '\n'.join(stdout_lines), '\n'.join(stderr_lines)
            
        except Exception as e:
            self.logger.error(f"监控进程输出失败: {e}")
            return '', str(e)
    
    def verify_conversion_results(self, test_data_dir: Path, expected_formats: List[str]) -> Dict:
        """验证转换结果"""
        results = {
            'success': True,
            'before_files': {},
            'after_files': {},
            'conversions': [],
            'errors': []
        }
        
        try:
            # 扫描转换后的文件
            after_files = self.scan_directory_files(test_data_dir)
            results['after_files'] = after_files
            
            # 检查是否有预期的格式转换
            for expected_format in expected_formats:
                if expected_format in after_files:
                    results['conversions'].append(f"成功转换为 {expected_format}")
                else:
                    results['errors'].append(f"未找到预期格式 {expected_format}")
                    results['success'] = False
            
            # 检查是否有异常文件（如临时文件、备份文件等）
            suspicious_patterns = ['.tmp', '.temp', '.bak', '.backup', '.pixly_']
            for pattern in suspicious_patterns:
                suspicious_files = [f for files in after_files.values() for f in files if pattern in f]
                if suspicious_files:
                    results['errors'].append(f"发现可疑文件: {suspicious_files}")
            
        except Exception as e:
            results['success'] = False
            results['errors'].append(f"验证转换结果失败: {e}")
        
        return results
    
    def run_single_test(self, test_config: Dict) -> Dict:
        """运行单个测试"""
        test_name = test_config['name']
        self.logger.info(f"开始测试: {test_name}")
        
        test_result = {
            'name': test_name,
            'start_time': datetime.now().isoformat(),
            'success': False,
            'errors': [],
            'details': {}
        }
        
        try:
            # 准备测试数据
            test_data_copy = self.prepare_test_data(test_name)
            
            # 记录转换前的文件状态
            before_files = self.scan_directory_files(test_data_copy)
            test_result['details']['before_files'] = before_files
            
            # 启动Pixly进程
            if not self.start_pixly_process():
                test_result['errors'].append("启动Pixly进程失败")
                return test_result
            
            # 等待界面加载
            time.sleep(2)
            
            # 模拟用户交互 - 输入目录路径
            self.send_input_to_process(str(test_data_copy))
            
            # 选择处理模式
            mode = test_config.get('mode', '1')  # 默认自动模式+
            self.send_input_to_process(mode)
            
            # 监控进程输出
            stdout, stderr = self.monitor_process_output(timeout=test_config.get('timeout', 300))
            test_result['details']['stdout'] = stdout
            test_result['details']['stderr'] = stderr
            
            # 等待处理完成
            time.sleep(5)
            
            # 验证转换结果
            verification = self.verify_conversion_results(
                test_data_copy, 
                test_config.get('expected_formats', [])
            )
            test_result['details']['verification'] = verification
            
            # 检查是否有错误
            if stderr and 'error' in stderr.lower():
                test_result['errors'].append(f"进程输出包含错误: {stderr}")
            
            if not verification['success']:
                test_result['errors'].extend(verification['errors'])
            
            # 检查质量判断引擎是否误报
            if '误报' in stdout or '全部损坏' in stdout:
                test_result['errors'].append("质量判断引擎可能出现误报")
            
            test_result['success'] = len(test_result['errors']) == 0
            
        except Exception as e:
            test_result['errors'].append(f"测试执行异常: {e}")
            self.logger.error(f"测试 {test_name} 执行异常: {e}")
            
        finally:
            # 清理进程
            if self.process:
                try:
                    self.process.terminate()
                    self.process.wait(timeout=10)
                except:
                    self.process.kill()
                self.process = None
            
            test_result['end_time'] = datetime.now().isoformat()
            self.logger.info(f"测试 {test_name} 完成，成功: {test_result['success']}")
            
        return test_result
    
    def generate_test_report(self):
        """生成测试报告"""
        report_file = self.results_dir / f"test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        
        # 更新汇总信息
        self.test_results['end_time'] = datetime.now().isoformat()
        self.test_results['summary']['total'] = len(self.test_results['tests'])
        self.test_results['summary']['passed'] = len([t for t in self.test_results['tests'] if t['success']])
        self.test_results['summary']['failed'] = self.test_results['summary']['total'] - self.test_results['summary']['passed']
        
        # 保存JSON报告
        with open(report_file, 'w', encoding='utf-8') as f:
            json.dump(self.test_results, f, ensure_ascii=False, indent=2)
        
        # 生成HTML报告
        html_report = self.generate_html_report()
        html_file = self.results_dir / f"test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.html"
        with open(html_file, 'w', encoding='utf-8') as f:
            f.write(html_report)
        
        self.logger.info(f"测试报告生成: {report_file}")
        self.logger.info(f"HTML报告生成: {html_file}")
        
        # 打印汇总
        summary = self.test_results['summary']
        self.logger.info(f"测试汇总 - 总计: {summary['total']}, 通过: {summary['passed']}, 失败: {summary['failed']}")
    
    def generate_html_report(self) -> str:
        """生成HTML测试报告"""
        html = f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Pixly UI自动化测试报告</title>
    <style>
        body {{ font-family: Arial, sans-serif; margin: 20px; }}
        .summary {{ background: #f5f5f5; padding: 15px; margin-bottom: 20px; border-radius: 5px; }}
        .test-case {{ border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }}
        .success {{ border-left: 5px solid #28a745; }}
        .failed {{ border-left: 5px solid #dc3545; }}
        .details {{ background: #f8f9fa; padding: 10px; margin-top: 10px; font-family: monospace; font-size: 12px; }}
        .error {{ color: #dc3545; }}
        pre {{ white-space: pre-wrap; word-wrap: break-word; }}
    </style>
</head>
<body>
    <h1>Pixly UI自动化测试报告</h1>
    
    <div class="summary">
        <h2>测试汇总</h2>
        <p>开始时间: {self.test_results['start_time']}</p>
        <p>结束时间: {self.test_results.get('end_time', 'N/A')}</p>
        <p>总计: {self.test_results['summary']['total']} | 
           通过: {self.test_results['summary']['passed']} | 
           失败: {self.test_results['summary']['failed']}</p>
    </div>
    
    <h2>测试详情</h2>
"""
        
        for test in self.test_results['tests']:
            status_class = 'success' if test['success'] else 'failed'
            status_text = '✅ 通过' if test['success'] else '❌ 失败'
            
            html += f"""
    <div class="test-case {status_class}">
        <h3>{test['name']} - {status_text}</h3>
        <p>开始时间: {test['start_time']}</p>
        <p>结束时间: {test.get('end_time', 'N/A')}</p>
        
        {''.join(f'<p class="error">错误: {error}</p>' for error in test['errors'])}
        
        <div class="details">
            <h4>详细信息:</h4>
            <pre>{json.dumps(test['details'], ensure_ascii=False, indent=2)}</pre>
        </div>
    </div>
"""
        
        html += """
</body>
</html>
"""
        return html


def main():
    """主函数"""
    # 配置路径
    pixly_binary = "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/pixly_test"
    test_data_dir = "/Users/nameko_1/Documents/Pixly/test_pack_all/不同格式测试合集_测试运行"
    results_dir = "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/tests/automation/results"
    
    # 创建自动化测试实例
    automation = PixlyUIAutomation(pixly_binary, test_data_dir, results_dir)
    
    # 设置测试环境
    if not automation.setup_test_environment():
        print("测试环境设置失败")
        return
    
    # 定义测试用例
    test_cases = [
        {
            'name': 'auto_mode_test',
            'mode': '1',  # 自动模式+
            'timeout': 180,
            'expected_formats': ['.jxl', '.avif'],
            'description': '测试自动模式+的转换功能'
        },
        {
            'name': 'quality_mode_test', 
            'mode': '2',  # 品质模式
            'timeout': 240,
            'expected_formats': ['.jxl', '.avif', '.mov'],
            'description': '测试品质模式的无损转换'
        },
        {
            'name': 'sticker_mode_test',
            'mode': '3',  # 表情包模式
            'timeout': 120,
            'expected_formats': ['.avif'],
            'description': '测试表情包模式的压缩功能'
        }
    ]
    
    # 运行测试
    print("开始自动化测试...")
    for test_config in test_cases:
        print(f"运行测试: {test_config['name']}")
        result = automation.run_single_test(test_config)
        automation.test_results['tests'].append(result)
        
        # 在测试间稍作休息
        time.sleep(5)
    
    # 生成报告
    automation.generate_test_report()
    print("自动化测试完成!")


if __name__ == "__main__":
    main()