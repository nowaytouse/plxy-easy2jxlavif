#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Pixly交互模拟器 - 简化版
通过stdin输入模拟用户交互，避免鼠标操作的复杂性
"""

import sys
import os
import subprocess
import time
import json
import shutil
import logging
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Tuple

class PixlyInteractionSimulator:
    """Pixly交互模拟器"""
    
    def __init__(self, pixly_binary: str, test_data_dir: str, results_dir: str):
        self.pixly_binary = Path(pixly_binary)
        self.test_data_dir = Path(test_data_dir)
        self.results_dir = Path(results_dir)
        
        # 创建结果目录
        self.results_dir.mkdir(parents=True, exist_ok=True)
        
        # 设置日志
        log_file = self.results_dir / f"simulation_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(log_file, encoding='utf-8'),
                logging.StreamHandler(sys.stdout)
            ]
        )
        self.logger = logging.getLogger(__name__)
        
        # 测试结果
        self.results = []
        
    def scan_files_by_format(self, directory: Path) -> Dict[str, int]:
        """按格式统计文件数量"""
        format_count = {}
        
        for file_path in directory.rglob('*'):
            if file_path.is_file() and not file_path.name.startswith('.'):
                suffix = file_path.suffix.lower()
                format_count[suffix] = format_count.get(suffix, 0) + 1
        
        return format_count
    
    def prepare_test_copy(self, test_name: str) -> Path:
        """准备测试数据副本"""
        copy_dir = self.results_dir / f"{test_name}_workspace"
        if copy_dir.exists():
            shutil.rmtree(copy_dir)
        
        shutil.copytree(self.test_data_dir, copy_dir)
        self.logger.info(f"准备测试副本: {copy_dir}")
        return copy_dir
    
    def run_pixly_with_inputs(self, test_copy: Path, inputs: List[str], timeout: int = 300) -> Tuple[str, str, int]:
        """运行Pixly并提供预设输入"""
        try:
            # 准备输入字符串
            input_data = '\n'.join(inputs) + '\n'
            
            self.logger.info(f"启动Pixly进程，输入: {inputs}")
            
            # 启动进程
            process = subprocess.Popen(
                [str(self.pixly_binary)],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                cwd=self.pixly_binary.parent
            )
            
            # 发送输入并等待完成
            stdout, stderr = process.communicate(input=input_data, timeout=timeout)
            return_code = process.returncode
            
            return stdout, stderr, return_code
            
        except subprocess.TimeoutExpired:
            process.kill()
            stdout, stderr = process.communicate()
            self.logger.error(f"进程超时被终止")
            return stdout, stderr, -1
            
        except Exception as e:
            self.logger.error(f"运行进程失败: {e}")
            return "", str(e), -2
    
    def analyze_logs_for_issues(self, stdout: str, stderr: str) -> List[str]:
        """分析日志中的问题"""
        issues = []
        
        # 检查stderr中的错误
        if stderr:
            if 'error' in stderr.lower():
                issues.append(f"标准错误输出包含错误: {stderr[:200]}...")
            if 'panic' in stderr.lower():
                issues.append(f"程序崩溃: {stderr[:200]}...")
        
        # 检查stdout中的异常情况
        stdout_lower = stdout.lower()
        
        # 检查质量判断引擎误报
        if '全部损坏' in stdout or '100%损坏' in stdout:
            issues.append("质量判断引擎可能误报所有文件为损坏")
        
        if '误报' in stdout:
            issues.append("日志中发现误报提及")
        
        # 检查扫描问题
        if '扫描失败' in stdout or '扫描错误' in stdout:
            issues.append("扫描环节出现问题")
        
        # 检查转换问题
        if '转换失败' in stdout_lower and '100%' in stdout:
            issues.append("所有文件转换失败，可能存在系统性问题")
        
        # 检查超时问题
        if '超时' in stdout or 'timeout' in stdout_lower:
            issues.append("处理过程中出现超时")
        
        # 检查内存问题
        if '内存' in stdout and ('不足' in stdout or '溢出' in stdout):
            issues.append("内存使用异常")
        
        return issues
    
    def verify_conversion_success(self, before_formats: Dict[str, int], after_formats: Dict[str, int], expected_formats: List[str]) -> Dict:
        """验证转换是否成功"""
        result = {
            'success': True,
            'conversions_found': [],
            'missing_formats': [],
            'format_changes': {},
            'issues': []
        }
        
        # 检查预期格式是否出现
        for expected in expected_formats:
            if expected in after_formats:
                result['conversions_found'].append(expected)
            else:
                result['missing_formats'].append(expected)
                result['success'] = False
        
        # 分析格式变化
        for format_name, count in before_formats.items():
            after_count = after_formats.get(format_name, 0)
            if after_count != count:
                result['format_changes'][format_name] = {
                    'before': count,
                    'after': after_count,
                    'change': after_count - count
                }
        
        # 检查是否有新格式出现
        new_formats = set(after_formats.keys()) - set(before_formats.keys())
        if new_formats:
            result['conversions_found'].extend(list(new_formats))
        
        # 检查可疑文件
        suspicious_formats = ['.tmp', '.temp', '.bak', '.backup']
        for suspicious in suspicious_formats:
            if suspicious in after_formats:
                result['issues'].append(f"发现可疑临时文件: {suspicious}")
        
        return result
    
    def run_test_scenario(self, test_config: Dict) -> Dict:
        """运行测试场景"""
        test_name = test_config['name']
        self.logger.info(f"开始测试场景: {test_name}")
        
        result = {
            'name': test_name,
            'start_time': datetime.now().isoformat(),
            'success': False,
            'issues': [],
            'details': {}
        }
        
        try:
            # 准备测试数据
            test_copy = self.prepare_test_copy(test_name)
            
            # 扫描原始文件
            before_formats = self.scan_files_by_format(test_copy)
            result['details']['before_formats'] = before_formats
            self.logger.info(f"转换前格式统计: {before_formats}")
            
            # 准备输入序列
            inputs = [
                str(test_copy),  # 目录路径
                test_config.get('mode', '1'),  # 处理模式
            ]
            
            # 添加额外的交互输入
            if 'additional_inputs' in test_config:
                inputs.extend(test_config['additional_inputs'])
            
            # 运行Pixly
            stdout, stderr, return_code = self.run_pixly_with_inputs(
                test_copy, 
                inputs, 
                test_config.get('timeout', 300)
            )
            
            result['details']['stdout'] = stdout
            result['details']['stderr'] = stderr
            result['details']['return_code'] = return_code
            
            # 分析日志问题
            log_issues = self.analyze_logs_for_issues(stdout, stderr)
            result['issues'].extend(log_issues)
            
            # 扫描转换后文件
            after_formats = self.scan_files_by_format(test_copy)
            result['details']['after_formats'] = after_formats
            self.logger.info(f"转换后格式统计: {after_formats}")
            
            # 验证转换结果
            conversion_result = self.verify_conversion_success(
                before_formats, 
                after_formats, 
                test_config.get('expected_formats', [])
            )
            result['details']['conversion_verification'] = conversion_result
            
            if not conversion_result['success']:
                result['issues'].extend(conversion_result['missing_formats'])
            
            # 检查返回码
            if return_code != 0:
                result['issues'].append(f"进程非正常退出，返回码: {return_code}")
            
            # 判断测试是否成功
            result['success'] = len(result['issues']) == 0
            
        except Exception as e:
            result['issues'].append(f"测试执行异常: {str(e)}")
            self.logger.error(f"测试 {test_name} 执行异常: {e}")
        
        result['end_time'] = datetime.now().isoformat()
        self.logger.info(f"测试 {test_name} 完成，成功: {result['success']}")
        
        if result['issues']:
            self.logger.warning(f"测试 {test_name} 发现问题: {result['issues']}")
        
        return result
    
    def generate_detailed_report(self):
        """生成详细的测试报告"""
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        
        # JSON报告
        json_report = {
            'timestamp': timestamp,
            'total_tests': len(self.results),
            'passed': len([r for r in self.results if r['success']]),
            'failed': len([r for r in self.results if not r['success']]),
            'tests': self.results
        }
        
        json_file = self.results_dir / f"detailed_report_{timestamp}.json"
        with open(json_file, 'w', encoding='utf-8') as f:
            json.dump(json_report, f, ensure_ascii=False, indent=2)
        
        # 生成可读性报告
        readable_report = self.generate_readable_report(json_report)
        text_file = self.results_dir / f"readable_report_{timestamp}.txt"
        with open(text_file, 'w', encoding='utf-8') as f:
            f.write(readable_report)
        
        self.logger.info(f"详细报告生成: {json_file}")
        self.logger.info(f"可读报告生成: {text_file}")
        
        return json_report
    
    def generate_readable_report(self, json_report: Dict) -> str:
        """生成可读性报告"""
        report = f"""
Pixly 自动化测试报告
生成时间: {json_report['timestamp']}
================================

测试汇总:
- 总测试数: {json_report['total_tests']}
- 通过: {json_report['passed']}
- 失败: {json_report['failed']}
- 成功率: {json_report['passed']/json_report['total_tests']*100:.1f}%

详细结果:
"""
        
        for test in json_report['tests']:
            status = "✅ 通过" if test['success'] else "❌ 失败"
            report += f"\n{test['name']} - {status}\n"
            report += f"  开始时间: {test['start_time']}\n"
            report += f"  结束时间: {test['end_time']}\n"
            
            if test['issues']:
                report += f"  发现问题:\n"
                for issue in test['issues']:
                    report += f"    - {issue}\n"
            
            # 格式转换信息
            details = test['details']
            if 'before_formats' in details and 'after_formats' in details:
                report += f"  格式变化:\n"
                report += f"    转换前: {details['before_formats']}\n"
                report += f"    转换后: {details['after_formats']}\n"
            
            if 'conversion_verification' in details:
                conv = details['conversion_verification']
                if conv['conversions_found']:
                    report += f"    发现转换: {conv['conversions_found']}\n"
                if conv['missing_formats']:
                    report += f"    缺失格式: {conv['missing_formats']}\n"
            
            report += "\n" + "="*50 + "\n"
        
        return report


def main():
    """主函数"""
    # 配置路径
    pixly_binary = "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/pixly_test"
    test_data_dir = "/Users/nameko_1/Documents/Pixly/test_pack_all/不同格式测试合集_测试运行"
    results_dir = "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/tests/automation/results"
    
    # 检查二进制文件是否存在
    if not Path(pixly_binary).exists():
        print(f"错误: Pixly二进制文件不存在: {pixly_binary}")
        print("请先构建项目: go build -o pixly_test .")
        return
    
    # 创建模拟器
    simulator = PixlyInteractionSimulator(pixly_binary, test_data_dir, results_dir)
    
    # 定义测试场景
    test_scenarios = [
        {
            'name': 'auto_mode_comprehensive',
            'mode': '1',  # 自动模式+
            'timeout': 180,
            'expected_formats': ['.jxl', '.avif'],
            'description': '自动模式+全面测试，检查智能路由和质量判断'
        },
        {
            'name': 'quality_mode_lossless',
            'mode': '2',  # 品质模式
            'timeout': 240,
            'expected_formats': ['.jxl', '.avif', '.mov'],
            'description': '品质模式测试，验证无损转换'
        },
        {
            'name': 'sticker_mode_compression',
            'mode': '3',  # 表情包模式
            'timeout': 120,
            'expected_formats': ['.avif'],
            'description': '表情包模式测试，验证极限压缩'
        },
        {
            'name': 'corrupted_files_handling',
            'mode': '1',
            'timeout': 60,
            'additional_inputs': ['4'],  # 如果检测到损坏文件，选择忽略
            'expected_formats': ['.jxl', '.avif'],
            'description': '测试损坏文件处理机制'
        }
    ]
    
    print("开始Pixly交互模拟测试...")
    
    # 运行所有测试
    for scenario in test_scenarios:
        print(f"\n运行测试: {scenario['name']}")
        result = simulator.run_test_scenario(scenario)
        simulator.results.append(result)
        
        # 简要输出结果
        status = "✅ 通过" if result['success'] else "❌ 失败"
        print(f"结果: {status}")
        if result['issues']:
            print(f"问题: {', '.join(result['issues'][:3])}...")
    
    # 生成报告
    print("\n生成详细报告...")
    report = simulator.generate_detailed_report()
    
    # 打印汇总
    print(f"\n测试完成!")
    print(f"总计: {report['total_tests']}, 通过: {report['passed']}, 失败: {report['failed']}")
    print(f"成功率: {report['passed']/report['total_tests']*100:.1f}%")


if __name__ == "__main__":
    main()