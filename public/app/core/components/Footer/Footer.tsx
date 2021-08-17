import React, { FC } from 'react';
import { Icon, IconName } from '@grafana/ui';

export interface FooterLink {
  text: string;
  icon?: string;
  url?: string;
  target?: string;
}

export let getFooterLinks = (): FooterLink[] => {
  return [
    {
      text: 'DeFiChain',
      icon: 'exchange-alt',
      url: 'https://defichain.com',
      target: '_blank',
    },

    {
      text: 'DeFiChain Wiki',
      icon: 'book-open',
      url: 'https://defichain-wiki.com',
      target: '_blank',
    },
    {
      text: 'DeFiChain Analytics',
      icon: 'database',
      url: 'https://defichain-analytics.com',
      target: '_blank',
    },
    {
      text: 'DefiChain Promo',
      icon: 'comments-alt',
      url: 'https://www.defichain-promo.com',
      target: '_blank',
    },
  ];
};

export let getVersionLinks = (): FooterLink[] => {
  const links: FooterLink[] = [];

  //links.push({ text: `${buildInfo.edition}${stateInfo}`, url: licenseInfo.licenseUrl });
  links.push({
    text: `DefiChain Explained`,
    url: 'https://www.defichain-explained.com',
  });

  return links;
};

export function setFooterLinksFn(fn: typeof getFooterLinks) {
  getFooterLinks = fn;
}

export function setVersionLinkFn(fn: typeof getFooterLinks) {
  getVersionLinks = fn;
}

export const Footer: FC = React.memo(() => {
  const links = getFooterLinks().concat(getVersionLinks());

  return (
    <footer className="footer">
      <div className="text-center">
        <ul>
          {links.map((link) => (
            <li key={link.text}>
              <a href={link.url} target={link.target} rel="noopener">
                {link.icon && <Icon name={link.icon as IconName} />} {link.text}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </footer>
  );
});

Footer.displayName = 'Footer';
