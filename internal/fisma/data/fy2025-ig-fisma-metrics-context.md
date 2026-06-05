# FY 2025 IG FISMA Metrics Evaluator's Guide — Context

**VERSION 1.0 | MAY 5, 2025**

This file captures the document-level guidance sections from the FY 2025 Inspector General FISMA Metrics Evaluator's Guide that apply across all metrics rather than to any individual one. The structured per-metric data (questions, maturity level descriptions, suggested evidence, criteria references, and assessor best practices) is in `fy2025-ig-fisma-metrics.json`.

---

## Introduction

To promote consistency in Inspectors General (IG) annual evaluations performed under the Federal Information Security Modernization Act of 2014 (FISMA), the Council of the Inspectors General on Integrity and Efficiency (CIGIE), in coordination with the Office of Management and Budget (OMB), the Department of Homeland Security (DHS), and the Federal Chief Information Officers and Chief Information Security Officers (CISO) councils are providing this evaluation guide for IGs to use in their FY 2025 FISMA evaluations.

The guide provides a baseline of suggested sources of evidence and test steps/objectives that can be used by IGs as part of their FISMA evaluations. The guide also includes suggested types of analysis that IGs may perform to assess capabilities in given areas. The guide should be considered for suggested source evidence that IGs may request to answer a metric. The guide should not be considered as an all-inclusive list of source evidence or test methods to reach the various maturity levels within metrics and domains. The test methods are not all inclusive and may not apply in all situations. Additional sources such as penetration testing and red team assessment results may be effective sources of evidence for select metrics.

The "Assessor's Best Practices" section has replaced the "Additional Notes" section this year. This section now breaks out the four maturity levels beyond Ad-Hoc to provide the assessor specific evaluation steps to consider for consistent assessment and testing. The steps provided are ones that have been used by experienced assessors and align to the maturity level and criteria for success.

The guide is a companion document to the FY 2025 IG FISMA Reporting Metrics (CISA) and OMB M-25-04, which provides guidance to IGs to assist in their FISMA evaluations.

---

## Determining Effectiveness with IG Metrics

IGs are required to assess the effectiveness of information security programs on a maturity model spectrum, in which the foundational levels ensure that agencies develop sound policies and procedures, and at the advanced levels capture the extent that agencies institutionalize those policies and procedures. The five maturity model levels are:

- **Level 1 — Ad Hoc**: No formal process defined or consistently applied.
- **Level 2 — Defined**: Policies and procedures exist.
- **Level 3 — Consistently Implemented**: Policies and procedures consistently applied across the organization.
- **Level 4 — Managed and Measurable**: Quantitative monitoring and reporting; OMB considers this the threshold for an effective security program.
- **Level 5 — Optimized**: Near real-time adaptation leveraging predictive analytics and threat intelligence.

Within the context of the maturity model, OMB believes that achieving Managed and Measurable (Level 4) or above represents an effective level of security. NIST SP 800-53 Rev. 5 provides additional guidance for determining the effectiveness of security controls. If an agency does not reach Level 4 or above for any metric, IGs are required to provide a summary in DHS's CyberScope portal as to why that metric only achieved Level 3 or below.

IGs should write Level 4 and its gaps in maturity. For example: "The Agency information security program is not effective because …." IGs should consider both their and the agency's assessment of unique missions, resources, and challenges when determining information security program effectiveness.

IGs have the discretion to determine whether an agency is effective in each of the Cybersecurity Framework Function (i.e., govern, identify, protect, detect, respond, and recover) and whether the agency's overall information security program is effective based on the results of the determinations of effectiveness in each function and the overall assessment. Therefore, an IG has the discretion to determine that an agency's information security program is effective even if the agency does not achieve Managed and Measurable (Level 4). Some agencies might uniquely meet these maturity levels, acknowledging the diverse nature of federal agencies' missions and resources.

Reflecting OMB's shift in emphasis away from compliance in favor of risk management-based security, IGs are encouraged to evaluate the IG metrics based on the risk tolerance and threat model of their agency and to focus on the practical security impact of weak control implementations, rather than strictly evaluating from a view of compliance or the mere presence or absence of controls. To facilitate this shift, and provide a foundation for assessing risk-based security objectives, starting in FY 2025, IGs are required to assess the extent to which agencies develop and maintain cybersecurity profiles that are used to understand, tailor, assess, prioritize, and communicate cybersecurity objectives.

---

## Core and Supplemental Metrics

In FY 2022, OMB implemented a framework regarding the timing and focus of assessments to provide a more flexible but continued focus on annual assessments for the federal community. This yielded two distinct groups of metrics.

### Core Metrics

There are 20 core metrics. The core metrics are assessed annually by the IGs and represent a combination of Administration priorities, high impact security processes, and essential functions necessary to determine security program effectiveness.

### Supplemental Metrics

Supplemental metrics are not considered core metrics but represent important activities conducted by security programs and contribute to the overall evaluation and determination of security program effectiveness. For FY 2025, the supplemental metrics comprise five new metrics designed to gauge the maturity of agencies' cybersecurity governance practices and implementation of key components of Zero Trust Architecture (ZTA). These five metrics will be evaluated by IGs and scored in FY 2025. IGs will consider the supplemental metric ratings when making the domain and function level maturity determinations.

---

## Terms

**Organization / Enterprise**: The terms are often used interchangeably. For the purposes of this document, an *organization* is defined as an entity of any size, complexity, or positioning within a larger organizational structure (e.g., a federal agency or department). An *enterprise* is an organization by this definition, but it exists at the top level of the hierarchy where individual senior leaders have unique risk management responsibilities (e.g., a federal agency or department). In terms of cybersecurity risk management (CSRM), most responsibilities tend to be carried out by individual organizations within an enterprise. In contrast, the responsibility for tracking key enterprise risks and their impacts on objectives is held by top-level corporate officers and board members who have fiduciary and reporting duties not performed anywhere else in the enterprise. (Reference: NISTIR 8286, Integrating Cybersecurity and Enterprise Risk Management.)

**Auditor / Assessor / Evaluator / IG / OIG**: The terms are often used interchangeably. The individuals performing the FISMA Metric reviews will vary from agency to agency. Some agencies have chosen to outsource the evaluation to contracted service providers.

**Information system / FISMA system / System**: The terms are often used interchangeably. For the purposes of FISMA and this document, an *information system* is a discrete set of information resources organized for the collection, processing, maintenance, use, sharing, dissemination, or disposition of information. According to FISMA, the head of Federal agencies are responsible for providing information security protections commensurate with the risk and magnitude of the harm resulting from unauthorized access, use, disclosure, disruption, modification, or destruction of information systems used or operated by their agency or on behalf of their agency by a contractor or other organization.

---

## Alternative Evidence Considerations

While the per-metric tables provide recommended types of evidence for evaluating maturity levels, IGs should consider accepting additional forms of evidence that effectively demonstrate capability maturity. The following alternative evidence approaches could complement traditional documentation:

1. **Demonstrated Capability**: Direct observation or demonstration of security capabilities functioning in actual operational environments.
2. **Results-oriented**: Data showing measurable improvements in security posture (e.g., reduction in incidents, faster response times).
3. **Performance Testing**: Results and actions taken to address findings from penetration tests, tabletop exercises, or security simulations.
4. **Continuous Monitoring Data**: Metrics and alerts from active monitoring systems.
5. **Adaptability**: Examples of how the agency has adjusted controls in response to emerging threats.
6. **Integration**: Demonstration of how controls work together as a cohesive system rather than isolated components.

These alternative forms of evidence may be particularly valuable when traditional documentation does not fully capture the effectiveness of an agency's security program. The intent of these suggestions is to support a holistic assessment approach that values security effectiveness alongside formal documentation.

---

## Recommendations Guidance

Although assessors have autonomy over what they feel is an appropriate recommendation for their organization, this section provides general guidance for consideration to make recommendations more consistent and effective across the Federal government.

### How to Write a Recommendation

Recommendations should be written from the perspective of what level the organization is at for the metric, and what it would take to progress to the next level. Broad recommendations should be avoided. Recommendations should be focused on specific actions to address the root cause and lead the agency to that next maturity level. It may require several recommendations to get that metric to the next level; however, this provides the agency with specific guidance and the opportunity to make steady and visible progress. This approach would also allow the assessors to follow up on agency actions taken as part of their recommendation follow-up processes and/or the next FISMA evaluation. Generally, a higher quantity of specific recommendations is preferable over fewer broad recommendations.

### Plans of Action and Milestones (POA&M)

As part of the data collection process, it is recommended that assessors collect and consider open POA&Ms that the organization has self-identified (or identified through other means, such as past GAO or OIG reports, or assessment and authorization reviews) as issues they are working to resolve. Assessors should avoid issuing recommendations that the organization is already actively working to resolve. Assessors should consider referencing open POA&Ms in the narrative write-up. Another approach is to issue an "Opportunity for Improvement" (OFI) or an "Item for Management's Consideration" (IMC) — unofficial recommendations that the assessor can issue in the report that are not tracked in monthly reports and semi-annual reports (SAR), but go on record to emphasize the issue. OFIs and IMCs could become recommendations over time (generally 1–2 years) if the POA&Ms or OFIs and IMCs are not timely resolved.

### Keeping Remediation Plans Current

OIGs and Agency officials ordinarily review and collaborate on recommendations to reach a management decision — either through the agency's official comments to the report or during recommendation follow-up. During the management decision process it is critical that all parties are clear and agree upon the agency's planned corrective action, ensuring it meets the intent of the recommendation and the selected NIST SP 800-53 controls (or other applicable criteria). Occasionally planned corrective actions may change due to recency and relevancy (time to fix, resources, change in technology, etc.); in these cases it is recommended that agency officials renegotiate the new planned corrective actions with OIG officials to develop an updated, agreed upon management decision.

### Recommendations Overcome by Events (OBE)

Technology and cyberspace are constantly and rapidly changing. A recommendation made today may quickly be OBE and no longer be feasible. Rather than leaving a recommendation open and trying to figure out how to address it, or simply closing it, OIGs should consider closing the recommendation with a status of "Unresolved – Closed," which records the fact that the agency was not able to address the issue. Then, if appropriate, an updated and refocused recommendation should be issued and go through the management decision process to help facilitate the agency's efforts to meet the OIG's original intent.

### Challenges from Reorganizations and Personnel Changes

Consistent with Government Auditing Standards (Yellow Book) and CIGIE's Quality Standards for Inspection and Evaluation (Blue Book), IGs should document the impact that scope limitations, restrictions on access to records, or other issues affecting their ability to complete FISMA reviews. Further, IGs should explain in CyberScope the impact this has on the IGs' ability to determine the effectiveness of their agency's information security program.
